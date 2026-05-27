package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AlexxIT/go2rtc/pkg/core"
	rtch264 "github.com/AlexxIT/go2rtc/pkg/h264"
	"github.com/AlexxIT/go2rtc/pkg/h264/annexb"
	rtc "github.com/AlexxIT/go2rtc/pkg/webrtc"
	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v4"
)

const (
	cameraPort = "8000"
)

type cameraFrame struct {
	data   []byte
	config string
}

type cameraHub struct {
	mu      sync.RWMutex
	clients map[*cameraClient]struct{}
	codec   atomic.Value
}

type cameraClient struct {
	conn *websocket.Conn
	ch   chan cameraFrame
}

func newCameraHub() *cameraHub {
	h := &cameraHub{clients: map[*cameraClient]struct{}{}}
	h.codec.Store("avc1.42E01F")
	return h
}

func (h *cameraHub) setCodec(codec string) {
	if codec != "" {
		h.codec.Store(codec)
		h.broadcastConfig(codec)
	}
}

func (h *cameraHub) add(c *cameraClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *cameraHub) remove(c *cameraClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.ch)
	_ = c.conn.Close()
}

func (h *cameraHub) broadcast(frame []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.ch <- cameraFrame{data: frame}:
		default:
		}
	}
}

func (h *cameraHub) broadcastConfig(codec string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.ch <- cameraFrame{config: codec}:
		default:
		}
	}
}

func (h *cameraHub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func startCameraServer() {
	var activeWS atomic.Int64

	mux := http.NewServeMux()
	mux.HandleFunc("/", cameraHandler(&activeWS))

	addr := ":" + cameraPort
	info("Camera relay ready on port %s", cameraPort)
	if err := http.ListenAndServe(addr, mux); err != nil {
		info("Camera relay stopped: %v", err)
	}
}

func runCameraReceiver(ctx context.Context, h *cameraHub) error {
	target := currentPrinterIP()
	if target == "" {
		return errors.New("printer ip is not discovered yet")
	}

	api, err := rtc.NewAPI()
	if err != nil {
		return err
	}

	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers:   nil,
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	})
	if err != nil {
		return err
	}
	defer pc.Close()

	done := make(chan error, 1)
	var pliSSRC atomic.Uint32

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		info("Camera peer connection state: %s", state)
		switch state {
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed, webrtc.PeerConnectionStateDisconnected:
			select {
			case done <- fmt.Errorf("peer connection %s", state):
			default:
			}
		}
	})

	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if track.Kind() != webrtc.RTPCodecTypeVideo || !strings.EqualFold(track.Codec().MimeType, webrtc.MimeTypeH264) {
			info("Ignoring camera track kind=%s codec=%s", track.Kind(), track.Codec().MimeType)
			return
		}

		pliSSRC.Store(uint32(track.SSRC()))
		codec := codecStringFromFMTP(track.Codec().SDPFmtpLine)
		h.setCodec(codec)
		info("Camera video track: payload=%d codec=%s fmtp=%q", track.PayloadType(), codec, track.Codec().SDPFmtpLine)

		depay := rtch264.RTPDepay(&core.Codec{
			Name:      core.CodecH264,
			ClockRate: 90000,
			FmtpLine:  track.Codec().SDPFmtpLine,
		}, makeCameraFrameHandler(h))

		for {
			packet, _, err := track.ReadRTP()
			if err != nil {
				select {
				case done <- err:
				default:
				}
				return
			}
			depay(packet)
		}
	})

	if _, err = pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeVideo,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	); err != nil {
		return err
	}

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return err
	}
	if err = pc.SetLocalDescription(offer); err != nil {
		return err
	}
	<-webrtc.GatheringCompletePromise(pc)

	body, err := offerToB64(pc.LocalDescription().SDP)
	if err != nil {
		return err
	}

	url := "http://" + net.JoinHostPort(target, cameraPort) + "/call/webrtc_local"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "plain/text")

	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("offer failed: %s: %s", res.Status, strings.TrimSpace(string(b)))
	}

	answer, err := answerFromB64(res.Body)
	if err != nil {
		return err
	}
	answer, err = fixCrealitySDP(answer)
	if err != nil {
		return err
	}

	if err = pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answer,
	}); err != nil {
		return err
	}

	pliTicker := time.NewTicker(2 * time.Second)
	defer pliTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			return err
		case <-pliTicker.C:
			ssrc := pliSSRC.Load()
			if ssrc != 0 && h.count() > 0 {
				_ = pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: ssrc}})
			}
		}
	}
}

func makeCameraFrameHandler(h *cameraHub) func(*rtp.Packet) {
	var firstTS uint32
	var haveFirst bool

	return func(packet *rtp.Packet) {
		if len(packet.Payload) == 0 {
			return
		}
		if !haveFirst {
			firstTS = packet.Timestamp
			haveFirst = true
		}

		annexB := annexb.DecodeAVCCWithAUD(packet.Payload)
		timestampUS := uint64(uint32(packet.Timestamp-firstTS)) * 1_000_000 / 90_000

		msg := make([]byte, 9+len(annexB))
		binary.BigEndian.PutUint64(msg[:8], timestampUS)
		if containsH264IDR(annexB) {
			msg[8] = 1
		}
		copy(msg[9:], annexB)

		h.broadcast(msg)
	}
}

func containsH264IDR(b []byte) bool {
	for i := 0; i+4 < len(b); i++ {
		if b[i] != 0 || b[i+1] != 0 {
			continue
		}
		offset := 0
		if b[i+2] == 1 {
			offset = 3
		} else if i+5 < len(b) && b[i+2] == 0 && b[i+3] == 1 {
			offset = 4
		}
		if offset == 0 || i+offset >= len(b) {
			continue
		}
		if b[i+offset]&0x1F == 5 {
			return true
		}
	}
	return false
}

func codecStringFromFMTP(fmtp string) string {
	const key = "profile-level-id="
	lower := strings.ToLower(fmtp)
	i := strings.Index(lower, key)
	if i < 0 {
		return "avc1.42E01F"
	}
	value := fmtp[i+len(key):]
	if j := strings.IndexAny(value, "; \t\r\n"); j >= 0 {
		value = value[:j]
	}
	if len(value) != 6 {
		return "avc1.42E01F"
	}
	return "avc1." + strings.ToUpper(value)
}

func offerToB64(sdp string) (string, error) {
	b, err := json.Marshal(map[string]string{
		"type": "offer",
		"sdp":  sdp,
	})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func answerFromB64(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	b, err = base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil {
		return "", err
	}
	var answer map[string]string
	if err = json.Unmarshal(b, &answer); err != nil {
		return "", err
	}
	if answer["sdp"] == "" {
		return "", errors.New("answer has no sdp")
	}
	return answer["sdp"], nil
}

func fixCrealitySDP(value string) (string, error) {
	var sd sdp.SessionDescription
	if err := sd.UnmarshalString(value); err != nil {
		return "", err
	}
	if len(sd.MediaDescriptions) == 0 {
		return value, nil
	}

	md := sd.MediaDescriptions[0]
	if len(md.MediaName.Formats) > 1 {
		skip := md.MediaName.Formats[0]
		md.MediaName.Formats = md.MediaName.Formats[1:]

		attrs := make([]sdp.Attribute, 0, len(md.Attributes))
		for _, attr := range md.Attributes {
			switch attr.Key {
			case "fmtp", "rtpmap":
				if strings.HasPrefix(attr.Value, skip) || strings.Contains(attr.Value, "x-google") {
					continue
				}
			}
			attrs = append(attrs, attr)
		}
		md.Attributes = attrs
	}

	b, err := sd.Marshal()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

var cameraUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func cameraHandler(activeWS *atomic.Int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !websocket.IsWebSocketUpgrade(r) {
			cameraIndexHandler(w, r)
			return
		}

		conn, err := cameraUpgrader.Upgrade(w, r, nil)
		if err != nil {
			info("Camera websocket upgrade: %v", err)
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		h := newCameraHub()
		c := &cameraClient{
			conn: conn,
			ch:   make(chan cameraFrame, 1),
		}
		activeWS.Add(1)
		defer activeWS.Add(-1)
		h.add(c)
		defer h.remove(c)

		go func() {
			for {
				if err := runCameraReceiver(ctx, h); err != nil && !errors.Is(err, context.Canceled) {
					info("Camera upstream stopped: %v", err)
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(3 * time.Second):
					info("Camera upstream reconnecting")
				}
			}
		}()

		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					cancel()
					_ = conn.Close()
					return
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case frame := <-c.ch:
				_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
				if frame.config != "" {
					cfg := map[string]string{
						"type":  "config",
						"codec": frame.config,
					}
					if err := conn.WriteJSON(cfg); err != nil {
						return
					}
				} else if err := conn.WriteMessage(websocket.BinaryMessage, frame.data); err != nil {
					return
				}
			}
		}
	}
}

func currentPrinterIP() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentIP
}

func cameraIndexHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, cameraIndexHTML)
}

//go:embed camera.html
var cameraIndexHTML string
