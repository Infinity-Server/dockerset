package main

import (
  "io"
  "os"
  "log"
  "net"
  "fmt"
  "bytes"
  "regexp"
  "net/url"
  "context"
  "io/ioutil"
  "net/http"
)

var (
  URL = os.Getenv("URL")
  STAGE = os.Getenv("STAGE")
  METHOD = os.Getenv("METHOD")
  SEARCH = os.Getenv("SEARCH")
  REPLACEMENT = os.Getenv("REPLACEMENT")

  FORWARD_SERVER = os.Getenv("FORWARD_SERVER")

  LISTEN_PORT = os.Getenv("LISTEN_PORT")
)

type Detail struct {
  url           *regexp.Regexp
  stage         string
  method        string
  search        *regexp.Regexp
  replacement   []byte
}

type RewriteBody struct {
  stage         string
  detail        Detail
  lastModified  bool
  next          http.Handler
}

func (r *RewriteBody) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
  if r.stage == "response" {
    r.ServeHTTPResponse(rw, req)
  } else {
    r.ServeHTTPRequest(rw, req)
  }
}

func (r *RewriteBody) ServeHTTPResponse(rw http.ResponseWriter, req *http.Request) {
  wrappedWriter := &ResponseWriter{
    lastModified:   r.lastModified,
    ResponseWriter: rw,
  }

  r.next.ServeHTTP(wrappedWriter, req)
  bodyBytes := wrappedWriter.buffer.Bytes()
  contentEncoding := wrappedWriter.Header().Get("Content-Encoding")
  if contentEncoding != "" && contentEncoding != "identity" {
    if _, err := rw.Write(bodyBytes); err != nil {
      log.Printf("unable to write body: %v", err)
    }
    return
  }

  rwt := r.detail
  if rwt.method != "" && rwt.url != nil {
    if rwt.method == req.Method && rwt.url.MatchString(req.URL.Path) {
      bodyBytes = rwt.search.ReplaceAll(bodyBytes, rwt.replacement)
    }
  } else {
    bodyBytes = rwt.search.ReplaceAll(bodyBytes, rwt.replacement)
  }

  if _, err := rw.Write(bodyBytes); err != nil {
    log.Printf("unable to write rewrited body: %v", err)
  }
}

func (r *RewriteBody) ServeHTTPRequest(rw http.ResponseWriter, req *http.Request) {
  bodyBytes, bodyErr := ioutil.ReadAll(req.Body)
  if bodyErr != nil {
    r.next.ServeHTTP(rw, req)
    return
  }
  req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

  rwt := r.detail
  if rwt.method != "" && rwt.url != nil {
    if rwt.method == req.Method && rwt.url.MatchString(req.URL.Path) {
      bodyBytes = rwt.search.ReplaceAll(bodyBytes, rwt.replacement)
    }
  } else {
    bodyBytes = rwt.search.ReplaceAll(bodyBytes, rwt.replacement)
  }

  req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
  req.ContentLength = int64(len(bodyBytes))
  r.next.ServeHTTP(rw, req)
}

type ResponseWriter struct {
  buffer       bytes.Buffer
  lastModified bool
  wroteHeader  bool
  http.ResponseWriter
}

func (r *ResponseWriter) WriteHeader(statusCode int) {
  if !r.lastModified {
    r.ResponseWriter.Header().Del("Last-Modified")
  }
  r.wroteHeader = true
  r.ResponseWriter.Header().Del("Content-Length")
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
  handler.stage = STAGE
  handler.detail.method = METHOD
  handler.detail.replacement = []byte(REPLACEMENT)

  url, err := regexp.Compile(URL)
  if err != nil {
    log.Println(err.Error())
    return
  }
  handler.detail.url = url

  search, err := regexp.Compile(SEARCH)
  if err != nil {
    log.Println(err.Error())
    return
  }
  handler.detail.search = search

  handler.next = &PassProxy{}

  http.HandleFunc("/", handler.ServeHTTP)
  errInfo := http.ListenAndServe(":" + LISTEN_PORT, nil)
  if err != nil {
    log.Println(errInfo.Error())
    return
  }
  log.Printf("[CHECKER] start http server on port %s ...\n", LISTEN_PORT)
}
