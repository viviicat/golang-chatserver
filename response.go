package main

import (
  "io"
  "bytes"
)


type Response struct {
  *bytes.Buffer
  Quit bool
}

func NewResponse(code string) Response {
  return Response{bytes.Buffer:bytes.NewBufferString(code+" ")}
}

func NewOkResponse() Response {
  return NewResponse("OK")
}

func NewMessageResponse(message *Message) Response {
  rs := NewResponse("FROM")
  rs.WriteString(message.from + " ")
  rs.Write(message.data)
  return rs
}

func NewErrorResponse(err error) Response {
  rs := NewResponse("ERROR")
  rs.WriteString(err.Error())
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

func (rs *Response) WriteTo(w io.Writer) (n int64, err error) {
  rs.WriteString("\r\n")
  return rs.Buffer.WriteTo(w)
}
