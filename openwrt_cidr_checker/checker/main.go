package main

import (
  "os"
  "fmt"
  "net"
  "strconv"
  "net/http"
  "encoding/json"
  curl "github.com/andelf/go-curl"
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

type AuthRequest struct {
  Id        int       `json:"id,omitempty"`
  Method    string    `json:"method,omitempty"`
  Params    []string  `json:"params,omitempty"`
}

type AuthResponse struct {
  Id        int       `json:"id,omitempty"`
  Result    string    `json:"result,omitempty"`
  Error     string    `json:"error,omitempty"`
}

func doFetch(api string, method string, data string) ([]byte, error) {
  easy := curl.EasyInit()
  defer easy.Cleanup()

  easy.Setopt(curl.OPT_URL, OPENWRT_HOST + api)
  easy.Setopt(curl.OPT_CUSTOMREQUEST, method)
  easy.Setopt(curl.OPT_FOLLOWLOCATION, 1);
  easy.Setopt(curl.OPT_DEFAULT_PROTOCOL, "http");
  easy.Setopt(curl.OPT_HTTPHEADER, []string{
    "Content-Type: application/json",
  })

  easy.Setopt(curl.OPT_POSTFIELDS, data);

  var response []byte
  onData := func (buf []byte, userdata interface{}) bool {
    response = append(response, buf...)
    return true
  }
  easy.Setopt(curl.OPT_WRITEFUNCTION, onData)

  if err := easy.Perform(); err != nil {
    return []byte{}, err
  }
  return response, nil
}

func doAuth() (*AuthResponse, error) {
  auth := AuthRequest{
    Id: 1,
    Method: "login",
    Params: []string{
      OPENWRT_USER,
      OPENWRT_PASS,
    },
  }

  body, err := json.Marshal(auth);
  if err != nil {
    return nil, err
  }

  data, err := doFetch("/cgi-bin/luci/rpc/auth", "GET", string(body))
  if err != nil {
    return nil, err
  }

  var response AuthResponse
  err = json.Unmarshal(data, &response)

  return &response, err
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

  data, err := doFetch("/ubus/", "POST", string(body))
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
        auth, err := doAuth()
        if err != nil {
          w.WriteHeader(statusFail)
          return
        }
        OPENWRT_ACCESS_TOKEN = auth.Result
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
    auth, err := doAuth()
    if err != nil {
      fmt.Printf("Error: %s", err.Error())
      os.Exit(-1)
    }
    OPENWRT_ACCESS_TOKEN = auth.Result
  }

  http.HandleFunc("/", httpHandler)
  err := http.ListenAndServe(":" + LISTEN_PORT, nil)
  if err != nil {
    fmt.Println(err.Error())
  }
}
