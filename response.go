package main

import (
  "fmt"
)


type Response struct {
  data []byte
  Quit bool
}

func NewResponse(code string) Response {
  data := append([]byte(code), ' ')
  return Response{data, false}
}

func NewOkResponse() Response {
  return NewResponse("OK")
}


func NewErrorResponse(err error) Response {
  rs := NewResponse("ERROR")
  rs.AppendString(err.Error())
  return rs
}

func NewFatalErrorResponse(err error) Response {
  rs := NewErrorResponse(err)
  rs.Quit = true
  return rs
}

func NewQuitResponse() Response {
  rs := NewResponse("TIUQ")
  rs.Quit = true
  return rs
}

func (rs *Response) AppendString(s string) {
  rs.Append([]byte(s))
}

func (rs *Response) Append(data []byte) {
  data = append(data, ' ')
  rs.data = append(rs.data, data...)
}

func (rs *Response) String() string {
  return string(rs.data)
}

func (rs *Response) WriteTo(c *Client) (n int, err error) {
  if *VerboseMode {
    fmt.Println("SENT to", c.String()+":", rs)
  }
  rs.data = append(rs.data, "\r\n"...)
  return c.Write(rs.data)
}

