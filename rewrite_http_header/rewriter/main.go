package main

import (
  "os"
  "fmt"
  "log"
  "regexp"
  "net/url"
  "net/http"
  "net/http/httputil"
)

var (
  URL = os.Getenv("URL")
  STAGE = os.Getenv("STAGE")
  METHOD = os.Getenv("METHOD")
  SEARCH = os.Getenv("SEARCH")
  HEADER = os.Getenv("HEADER")
  REPLACEMENT = os.Getenv("REPLACEMENT")

  FORWARD_SERVER = os.Getenv("FORWARD_SERVER")

  LISTEN_PORT = os.Getenv("LISTEN_PORT")

  detail Detail 
)

type Detail struct {
  url           *regexp.Regexp
  stage         string
  method        string
  header        string
  search        *regexp.Regexp
  replacement   []byte
}

func NewProxy(targetHost string) (*httputil.ReverseProxy, error) {
  url, err := url.Parse(targetHost)
  if err != nil {
    return nil, err
  }

  proxy := httputil.NewSingleHostReverseProxy(url)

  originalDirector := proxy.Director
  proxy.Director = func(req *http.Request) {
    originalDirector(req)
    modifyRequest(req)
  }

  proxy.ModifyResponse = modifyResponse
  proxy.ErrorHandler = errorHandler
  return proxy, nil
}

func modifyRequest(req *http.Request) {
  if STAGE != "request" {
    return
  }

  rwt := detail
  if rwt.method != "" && rwt.url != nil {
    if rwt.method == req.Method && rwt.url.MatchString(req.URL.Path) {
      if req.Header.Get(rwt.header) != "" {
        oldValues := req.Header.Values(rwt.header)
        values := make([]string, len(oldValues))
        copy(values, oldValues)
        req.Header.Del(rwt.header)
        for _, value := range values {
          bytesValue := []byte(value)
          newValue := string(rwt.search.ReplaceAll(bytesValue, rwt.replacement))
          req.Header.Add(rwt.header, newValue)
        }
      }
    }
  }
}
func modifyResponse(res *http.Response) error {
  if STAGE != "response" {
    return nil
  }

  rwt := detail
  if rwt.method != "" && rwt.url != nil {
    if rwt.method == res.Request.Method && rwt.url.MatchString(res.Request.URL.Path) {
      if res.Header.Get(rwt.header) != "" {
        oldValues := res.Header.Values(rwt.header)
        values := make([]string, len(oldValues))
        copy(values, oldValues)
        res.Header.Del(rwt.header)
        for _, value := range values {
          bytesValue := []byte(value)
          newValue := string(rwt.search.ReplaceAll(bytesValue, rwt.replacement))
          res.Header.Add(rwt.header, newValue)
        }
      }
    }
  }

  return nil
}

func errorHandler(w http.ResponseWriter, req *http.Request, err error) {
  fmt.Printf("Got error while modifying response: %v \n", err)
  return
}

func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
  return func(w http.ResponseWriter, r *http.Request) {
    proxy.ServeHTTP(w, r)
  }
}

func main() {
  detail = Detail{}

  detail.stage = STAGE
  detail.method = METHOD
  detail.header = HEADER
  detail.replacement = []byte(REPLACEMENT)

  url, err := regexp.Compile(URL)
  if err != nil {
    log.Println(err.Error())
    return
  }
  detail.url = url

  search, err := regexp.Compile(SEARCH)
  if err != nil {
    log.Println(err.Error())
    return
  }
  detail.search = search

  proxy, err := NewProxy("http://" + FORWARD_SERVER)
  if err != nil {
    panic(err)
  }

  http.HandleFunc("/", ProxyRequestHandler(proxy))
  errInfo := http.ListenAndServe(":" + LISTEN_PORT, nil)
  if err != nil {
    log.Println(errInfo.Error())
    return
  }
  log.Printf("[CHECKER] start http server on port %s ...\n", LISTEN_PORT)
}

