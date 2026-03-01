package main

import (
	"encoding/binary"
	"math/rand"
	"testing"
)

func buildQuery(name string, qtype uint16) []byte {
	msg := make([]byte, 12)
	binary.BigEndian.PutUint16(msg[0:2], 0x1234)
	binary.BigEndian.PutUint16(msg[2:4], 0x0000)
	binary.BigEndian.PutUint16(msg[4:6], 1)
	enc, _ := encodeName(name)
	msg = append(msg, enc...)
	msg = append(msg, byte(qtype>>8), byte(qtype))
	msg = append(msg, 0x00, 0x01)
	return msg
}

func TestParseIntentForA(t *testing.T) {
	pkt := buildQuery("host.local", 1)
	_, intent, err := parseIntent(pkt, "host.local")
	if err != nil {
		t.Fatalf("parseIntent returned error: %v", err)
	}
	if !intent.WantA || intent.WantAAAA {
		t.Fatalf("unexpected intent: %+v", intent)
	}
}

func TestParseIntentTruncated(t *testing.T) {
	cases := [][]byte{
		{},
		{0, 1, 2},
		{0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0},
		buildQuery("host.local", 1)[:14],
	}
	for i, pkt := range cases {
		if _, _, err := parseIntent(pkt, "host.local"); err == nil {
			t.Fatalf("case %d expected error", i)
		}
	}
}

func TestParseIntentNoPanicOnRandomData(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	for i := 0; i < 2000; i++ {
		n := r.Intn(600)
		pkt := make([]byte, n)
		_, _ = r.Read(pkt)
		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("panic for n=%d: %v", n, p)
				}
			}()
			_, _, _ = parseIntent(pkt, "host.local")
		}()
	}
}
