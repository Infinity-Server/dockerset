package main

import (
  "io"
  "os"
  "log"
  "net"
  "fmt"
  "bytes"
  "net/url"
  "context"
  "net/http"
  "io/ioutil"
  "path/filepath"
  "encoding/json"
  "compress/gzip"
  "github.com/tidwall/sjson"
)

var (
  LISTEN_PORT = os.Getenv("LISTEN_PORT")
  FORWARD_SERVER = os.Getenv("FORWARD_SERVER")
)

type RewriteBody struct {
  lastModified  bool
  next          http.Handler
}

type NavidromeSongResponse struct {
  Lyrics            string    `json:"lyrics,omitempty"`
  Path              string    `json:"path,omitempty"`
}

func (r *RewriteBody) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
  r.ServeHTTPResponse(rw, req)
}

func (r *RewriteBody) ServeHTTPResponse(rw http.ResponseWriter, req *http.Request) {
  wrappedWriter := &ResponseWriter{
    lastModified:   r.lastModified,
    ResponseWriter: rw,
  }

  r.next.ServeHTTP(wrappedWriter, req)
  bodyBytes := wrappedWriter.buffer.Bytes()
  contentEncoding := wrappedWriter.originalEncoding
  if contentEncoding == "gzip" {
    gzipReader, err := gzip.NewReader(bytes.NewReader(bodyBytes))
    if err != nil {
      log.Printf("unable decode gzip: %v", err)
      if _, err := rw.Write(bodyBytes); err != nil {
        log.Printf("unable to write body: %v", err)
      }
      return
    }
    result, _ := ioutil.ReadAll(gzipReader)
    bodyBytes = result
  }

  var body NavidromeSongResponse
  err := json.Unmarshal(bodyBytes, &body)
  if err != nil {
    log.Printf("unable to unmarshal navidrome response: %v", err)
    if _, err := rw.Write(bodyBytes); err != nil {
      log.Printf("unable to write body: %v", err)
    }
    return
  }

  lrcPath := fmt.Sprintf("%s.%s", body.Path[:len(body.Path) - len(filepath.Ext(body.Path))], "lrc")
  data, err := os.ReadFile(lrcPath)
  if err == nil {
    jsonData, _ := sjson.Set(string(bodyBytes), "lyrics", string(data))
    bodyBytes = []byte(jsonData)
  } else {
    log.Printf("unable to fine lrc file: %s", lrcPath)
  }

  if _, err := rw.Write(bodyBytes); err != nil {
    log.Printf("unable to write rewrited body: %v", err)
  }
}

type ResponseWriter struct {
  originalEncoding    string
  buffer              bytes.Buffer
  lastModified        bool
  wroteHeader         bool
  http.ResponseWriter
}

func (r *ResponseWriter) WriteHeader(statusCode int) {
  r.originalEncoding = r.ResponseWriter.Header().Get("Content-Encoding")
  if !r.lastModified {
    r.ResponseWriter.Header().Del("Last-Modified")
  }
  r.wroteHeader = true
  r.ResponseWriter.Header().Del("Content-Length")
  r.ResponseWriter.Header().Del("Content-Encoding")
  r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseWriter) Write(p []byte) (int, error) {
  if !r.wroteHeader {
    r.WriteHeader(http.StatusOK)
  }

  return r.buffer.Write(p)
}

func (r *ResponseWriter) Flush() {
  if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
    flusher.Flush()
  }
}

type PassProxy struct {
  forwardServer     string
}

func (pp PassProxy) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
  return net.Dial("tcp", FORWARD_SERVER)
}

func (pp PassProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  transport := http.Transport{
    Proxy: http.ProxyFromEnvironment,
    DialContext: pp.DialContext,
  }

  client := http.Client{
    Transport: &transport,
  }

  url, err := url.Parse(fmt.Sprintf("http://%s%s", r.Host, r.URL.Path))
  if err != nil {
    w.Write([]byte(err.Error()))
    return
  }

  r.URL = url
  r.RequestURI = ""

  res, err := client.Do(r)
  if err != nil {
    w.Write([]byte(err.Error()))
    return
  }

  for header, contents := range res.Header {
    w.Header().Del(header)
    for _, content := range contents {
      w.Header().Add(header, content)
    }
  }

  w.WriteHeader(res.StatusCode)

  io.Copy(w, res.Body)
  if err := res.Body.Close(); err != nil {
    w.Write([]byte(err.Error()))
  }
}

func main() {
  handler := &RewriteBody{}
  handler.next = &PassProxy{}

  http.HandleFunc("/", handler.ServeHTTP)
  err := http.ListenAndServe(":" + LISTEN_PORT, nil)
  if err != nil {
    log.Println(err.Error())
    return
  }
  log.Printf("[CHECKER] start http server on port %s ...\n", LISTEN_PORT)
}
