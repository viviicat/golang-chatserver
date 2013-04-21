package main

import (
  "net"
  "bytes"
)

type Respondable interface {
  Send(conn net.Conn) error
}

type Response struct {
  data []byte
}

func (rs *Response) Send(conn net.Conn) error {
  conn.Write(rs.data)
  conn.Write([]byte("\r\n"))
  return nil
}

func NewTahcResponse() Response {
  return Response{[]byte("TAHC")}
}

func NewOkResponse() Response {
  return Response{[]byte("OK")}
}

func NewMessageResponse(message *Message) Response {
  return Response{bytes.Join([][]byte{[]byte("FROM"), []byte(message.from), message.data}, []byte(" "))}
}

func NewErrorResponse(err error) Response {
  return Response{[]byte("ERROR " + err.Error())}
}

func NewUsersResponse(clients *List) Response {
  var data string
  for e := clients.Front(); e != nil; e = e.Next() {
    data += e.Value.(*Client).username + " "
  }
  return Response{[]byte("USERS " + data)}
}

func NewRoomsResponse(channels *map[string] *List) Response {
  var data string
  for key, _ := range *channels {
    data += key + " "
  }

  return Response{[]byte("ROOMS " + data)}
}


type QuitResponse struct {
  Response
}

func NewDisconnectResponse(err error) QuitResponse {
  return QuitResponse{Response:NewErrorResponse(err)}
}

func NewQuitResponse() QuitResponse {
  return QuitResponse{Response:Response{[]byte("TIUQ")}}
}

