package main

import (
  "os"
  "fmt"
  "net"
  "net/url"
  "strconv"
  "strings"
  "net/http"
  "io/ioutil"
  "encoding/json"
  iprange "github.com/netdata/go.d.plugin/pkg/iprange"
)

var (
  OPENWRT_HOST = os.Getenv("OPENWRT_HOST")
  OPENWRT_USER = os.Getenv("OPENWRT_USER")
  OPENWRT_PASS = os.Getenv("OPENWRT_PASS")

  OPENWRT_IFACE = os.Getenv("OPENWRT_IFACE")

  LISTEN_PORT = os.Getenv("LISTEN_PORT")

  STATUS_OK = os.Getenv("STATUS_OK")
  STATUS_FAIL = os.Getenv("STATUS_FAIL")

  OPENWRT_ACCESS_TOKEN = ""
)

const (
  FailedCountLimit    = 3
  AccessDeniedErrMsg  = "Access denied"
)

func encodeURIComponent(str string) string {
  r := url.QueryEscape(str)
  r = strings.Replace(r, "+", "%20", -1)
  return r
}

func doFetch(api string, contentType string, data string) ([]byte, error) {
  res, err := http.Post(OPENWRT_HOST + api, contentType, strings.NewReader(data))
  if err != nil {
    return []byte{}, err
  }

  defer res.Body.Close()
  body, err := ioutil.ReadAll(res.Body)
  if err != nil {
    return []byte{}, err
  }

  return body, nil
}

func doAuth() (string, error) {
  client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
      return http.ErrUseLastResponse
    },
  }

  params := url.Values{
    "luci_username": {OPENWRT_USER},
    "luci_password": {OPENWRT_PASS},
  }

  req, err := http.NewRequest("POST", OPENWRT_HOST + "/cgi-bin/luci/", strings.NewReader(params.Encode()))
  if err != nil {
    return "", err
  }

  req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

  res, err := client.Do(req)
  if err != nil {
    return "", err
  }

  for _, cookie := range res.Cookies() {
    if cookie.Name == "sysauth_http" {
      return cookie.Value, nil
    }
  }

  return "", fmt.Errorf("no cookie")
}

func getCIDR() (string, string, error) {
  CallArgsOne := "network.interface"
  CallArgsTwo := "dump"
  request := Welcome{
    WelcomeElement{
      ID: 10086,
      Jsonrpc: "2.0",
      Method: "call",
      Params: []ParamElement{
        {
          String: &OPENWRT_ACCESS_TOKEN,
        },
        {
          String: &CallArgsOne,
        },
        {
          String: &CallArgsTwo,
        },
        {
          ParamClass: &ParamClass{},
        },
      },
    },
  }

  body, err := json.Marshal(request);
  if err != nil {
    return "", "", err
  }

  data, err := doFetch("/ubus/", "application/json", string(body))
  if err != nil {
    return "", "", err
  }

  var response Welcome
  err = json.Unmarshal(data, &response)
  if err != nil {
    return "", "", nil
  }

  ipv4 := ""
  ipv6 := ""

  for _, w := range response {
    if w.Error != nil {
      return "", "", fmt.Errorf(w.Error.Message)
    }
    for _, r := range w.Result {
      if r.RealResult != nil {
        for _, i := range r.RealResult.Interface {
          if i.Interface == OPENWRT_IFACE {
            ipv4 = fmt.Sprintf("%s/%d", i.Ipv4Address[0].Address, i.Ipv4Address[0].Mask)
          }
          if len(i.Ipv6Prefix) > 0 {
            for _, prefix := range i.Ipv6Prefix {
              for k, v := range prefix.Assigned {
                if k == OPENWRT_IFACE {
                  ipv6 = fmt.Sprintf("%s/%d", v.Address, v.Mask)
                }
              }
            }
          }
        }
      }
    }
  }
  return ipv4, ipv6, nil
}

func httpHandler(w http.ResponseWriter, req *http.Request) {
  statusFail, _ := strconv.Atoi(STATUS_FAIL)

  realIP := req.Header.Get("x-real-ip")
  if realIP == "" {
    w.WriteHeader(statusFail)
    return
  }

  ip := net.ParseIP(realIP)
  if ip == nil {
    w.WriteHeader(statusFail)
    return
  }

  ipv4 := ""
  ipv6 := ""
  failedCount := 0
  for failedCount < FailedCountLimit {
    _ipv4, _ipv6, err := getCIDR()
    ipv4 = _ipv4
    ipv6 = _ipv6
    if err != nil {
      if err.Error() == AccessDeniedErrMsg {
        failedCount = failedCount + 1
        token, err := doAuth()
        if err != nil {
          w.WriteHeader(statusFail)
          return
        }
        OPENWRT_ACCESS_TOKEN = token
      } else {
        w.WriteHeader(statusFail)
        return
      }
    } else {
      break
    }
  }

  ipv4Range, err := iprange.ParseRange(ipv4)
  if err != nil {
    w.WriteHeader(statusFail)
    return
  }

  ipv6Range, err := iprange.ParseRange(ipv6)
  if err != nil {
    w.WriteHeader(statusFail)
    return
  }

  if ipv4Range.Contains(ip) || ipv6Range.Contains(ip) {
    status, _ := strconv.Atoi(STATUS_OK)
    w.WriteHeader(status)
    return
  }

  w.WriteHeader(statusFail)
}

func main() {
  if OPENWRT_ACCESS_TOKEN == "" {
    token, err := doAuth()
    if err != nil {
      fmt.Printf("Error: %s", err.Error())
      os.Exit(-1)
    }
    OPENWRT_ACCESS_TOKEN = token
  }

  http.HandleFunc("/", httpHandler)
  err := http.ListenAndServe(":" + LISTEN_PORT, nil)
  if err != nil {
    fmt.Println(err.Error())
  }
}
