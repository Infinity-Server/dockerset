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
  STATUS_OK_CODE = 0
  STATUS_FAIL_CODE = 0
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
  fmt.Printf("[CHECKER] get access token from openwrt ...\n")
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
      fmt.Printf("[CHECKER] openwrt sysauth_http=%s  ...\n", cookie.Value)
      return cookie.Value, nil
    }
  }

  return "", fmt.Errorf("no cookie")
}

func getCIDR() ([]string, []string, error) {
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
    return []string{}, []string{}, err
  }

  data, err := doFetch("/ubus/", "application/json", string(body))
  if err != nil {
    return []string{}, []string{}, err
  }

  var response Welcome
  err = json.Unmarshal(data, &response)
  if err != nil {
    return []string{}, []string{}, nil
  }

  var ipv4 []string
  var ipv6 []string

  for _, w := range response {
    if w.Error != nil {
      return []string{}, []string{}, fmt.Errorf(w.Error.Message)
    }
    for _, r := range w.Result {
      if r.RealResult != nil {
        for _, i := range r.RealResult.Interface {
          if i.Interface == OPENWRT_IFACE {
            for _, addr := range i.Ipv4Address {
              cidr := fmt.Sprintf("%s/%d", addr.Address, addr.Mask)
              ipv4 = append(ipv4, cidr)
            }
            for _, addr := range i.Ipv6PrefixAssignment {
              cidr := fmt.Sprintf("%s/%d", addr.Address, addr.Mask)
              ipv6 = append(ipv4, cidr)
            }
          }
        }
      }
    }
  }
  return ipv4, ipv6, nil
}

func httpHandler(w http.ResponseWriter, req *http.Request) {
  realIP := req.Header.Get("x-real-ip")
  if realIP == "" {
    w.WriteHeader(STATUS_FAIL_CODE)
    return
  }
  fmt.Printf("[CHECKER] receive new request from ip=%s ...\n", realIP)

  ip := net.ParseIP(realIP)
  if ip == nil {
    w.WriteHeader(STATUS_FAIL_CODE)
    return
  }

  var ipv4 []string
  var ipv6 []string
  failedCount := 0
  for failedCount < FailedCountLimit {
    _ipv4, _ipv6, err := getCIDR()
    ipv4 = _ipv4
    ipv6 = _ipv6
    if err != nil {
      if err.Error() == AccessDeniedErrMsg {
        failedCount = failedCount + 1
        fmt.Printf("[CHECKER] openwrt access denied, retry count=%d ...\n", failedCount)
        token, err := doAuth()
        if err != nil {
          w.WriteHeader(STATUS_FAIL_CODE)
          return
        }
        OPENWRT_ACCESS_TOKEN = token
      } else {
        w.WriteHeader(STATUS_FAIL_CODE)
        return
      }
    } else {
      break
    }
  }

  var ipRanges []iprange.Range
  for _, cidr := range append(ipv4, ipv6...) {
    ipRange, err := iprange.ParseRange(cidr)
    if err != nil {
      fmt.Printf("[CHECKER] parse failed for cidr=%s", ip)
    } else {
      ipRanges = append(ipRanges, ipRange)
    }
  }

  for _, ipRange := range ipRanges {
    if ipRange.Contains(ip) {
      w.WriteHeader(STATUS_OK_CODE)
      return
    }
  }

  w.WriteHeader(STATUS_FAIL_CODE)
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

  statusOk, _ := strconv.Atoi(STATUS_OK)
  statusFail, _ := strconv.Atoi(STATUS_FAIL)
  STATUS_OK_CODE = statusOk
  STATUS_FAIL_CODE = statusFail

  http.HandleFunc("/", httpHandler)
  err := http.ListenAndServe(":" + LISTEN_PORT, nil)
  if err != nil {
    fmt.Println(err.Error())
    return
  }
  fmt.Printf("[CHECKER] start http server on port %s ...\n", LISTEN_PORT)
}
