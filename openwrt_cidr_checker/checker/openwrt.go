package main

import (
  "encoding/json"
)

type Welcome []WelcomeElement

type ParamClass struct {
}

type ParamElement struct {
  ParamClass *ParamClass
  String     *string
}

func (x *ParamElement) MarshalJSON() ([]byte, error) {
  if x.String != nil {
    return json.Marshal(x.String)
  }
  return json.Marshal(x.ParamClass)
}

type JsonError struct {
  Code      int64         `json:"code,omitempty"`
  Message   string        `json:"message,omitempty"`
}

type WelcomeElement struct {
  Jsonrpc string          `json:"jsonrpc"`
  ID      int64           `json:"id"`     
  Result  []Result        `json:"result,omitempty"` 
  Method  string          `json:"method,omitempty"` 
  Params  []ParamElement  `json:"params,omitempty"` 
  Error   *JsonError      `json:"error,omitempty"`
}

type Result struct {
  RealResult      *RealResult
  Int             *int64
}

type RealResult struct {
  Interface   []Interface
}

type Interface struct {
  Interface            string        `json:"interface,omitempty"`             
  Ipv4Address          []Ipv4Address `json:"ipv4-address,omitempty"`          
  Ipv6Address          []Ipv6        `json:"ipv6-address,omitempty"`          
  Ipv6Prefix           []Ipv6Prefix  `json:"ipv6-prefix,omitempty"`           
  Ipv6PrefixAssignment []Ipv6        `json:"ipv6-prefix-assignment,omitempty"`
}

func (x *Result) UnmarshalJSON(data []byte) error {
  x.RealResult = nil
  var obj RealResult
  json.Unmarshal(data, &obj)
  x.RealResult = &obj
  return nil
}

type Ipv4Address struct {
  Address    string  `json:"address,omitempty"`             
  Mask       int64   `json:"mask,omitempty"`                
}

type Ipv6 struct {
  Address      string        `json:"address,omitempty"`                
  Mask         int64         `json:"mask,omitempty"`                   
}

type LocalAddress struct {
  Address string `json:"address,omitempty"`
  Mask    int64  `json:"mask,omitempty"`   
}

type Ipv6Prefix struct {
  Address   string   `json:"address,omitempty"`  
  Mask      int64    `json:"mask,omitempty"`     
  Preferred int64    `json:"preferred,omitempty"`
  Valid     int64    `json:"valid,omitempty"`    
  Class     string   `json:"class,omitempty"`    
  Assigned  Assigned `json:"assigned,omitempty"` 
}

type Assigned map[string] LocalAddress

type Route struct {
  Target  string `json:"target,omitempty"`          
  Mask    int64  `json:"mask,omitempty"`            
  Nexthop string `json:"nexthop,omitempty"`         
  Source  string `json:"source,omitempty"`          
}
