package main

import (
  "io"
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

func (rs *Response) WriteTo(w io.Writer) (n int, err error) {
  rs.data = append(rs.data, "\r\n"...)
  return w.Write(rs.data)
}

