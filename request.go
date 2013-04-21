package main

import (
  "bytes"
  "errors"
)

type Requestable interface {
  Create(buf []byte) error
  Handle(dispatcher *Dispatcher) (Respondable, error)
  SetClient(client *Client)
  GetClient() *Client
}

//////////////////////////////////////////////////

type Request struct {
  client *Client
}


func (rq *Request) Create(buf []byte) error {
  return nil
}

func (rq *Request) Handle(dispatcher *Dispatcher) (Respondable, error) {
  rs := NewTahcResponse()
  return &rs, nil
}

func (rq *Request) SetClient(client *Client) {
  rq.client = client
}

func (rq *Request) GetClient() *Client {
  return rq.client
}

//////////////////////////////////////////////////

type UserRequest struct {
  Request
  id ClientId
}

func (rq *UserRequest) Create(buf []byte) error {
  if rq.client.loggedIn {
    return errors.New("You are already logged in")
  }

  args := bytes.SplitN(buf, []byte(" "), 2)
  if len(args) != 2 {
    return errors.New("Invalid User Request")
  }

  var err error
  rq.id, err = NewClientId(string(args[0]), args[1])
  if err != nil {
    return err
  }

  return nil
}

func (rq *UserRequest) Handle(dispatcher *Dispatcher) (Respondable, error) {
  if err := dispatcher.ClientLogin(rq.client, rq.id.username, rq.id.password); err != nil {
    var rs Respondable
    if _, ok := err.(*DisconnectError); ok {
      dr := NewDisconnectResponse(err)
      rs = &dr
    } else {
      er := NewErrorResponse(err)
      rs = &er
    }
    return rs, err
  }

  rs := NewOkResponse()
  return &rs, nil
}

//////////////////////////////////////////////////

type UsersRequest struct {
  Request
}

func (rq *UsersRequest) Handle(dispatcher *Dispatcher) (Respondable, error) {
  rs := NewUsersResponse(dispatcher.clients)
  return &rs, nil
}

//////////////////////////////////////////////////

type RoomsRequest struct {
  Request
}

func (rq *RoomsRequest) Handle(dispatcher *Dispatcher) (Respondable, error) {
  rs := NewRoomsResponse(&dispatcher.channels)
  return &rs, nil
}

//////////////////////////////////////////////////

type JoinRequest struct {
  Request
  channel string
}

func (rq *JoinRequest) Create(buf []byte) error {
  rq.channel = string(buf)
  return nil
}

func (rq *JoinRequest) Handle(dispatcher *Dispatcher) (Respondable, error) {
  dispatcher.ClientJoin(rq.client, rq.channel)

  rs := NewOkResponse()
  return &rs, nil
}

//////////////////////////////////////////////////

type PartRequest struct {
  JoinRequest
}

func (rq *PartRequest) Handle(dispatcher *Dispatcher) (Respondable, error) {
  dispatcher.ClientPart(rq.client, rq.channel)

  rs := NewOkResponse()
  return &rs, nil
}

//////////////////////////////////////////////////

type Message struct {
  from string
  target string
  data []byte
}

type SayRequest struct {
  Request
  Message
}

func (rq *SayRequest) Create(buf []byte) error {
  spl := bytes.SplitN(buf, []byte(" "), 2)
  rq.from = rq.client.username
  rq.target = string(spl[0])
  rq.data = spl[1]
  return nil
}

func (rq *SayRequest) Handle(dispatcher *Dispatcher) (Respondable, error) {
  dispatcher.ClientSayTo(rq.client, &rq.Message)

  rs := NewOkResponse()
  return &rs, nil
}

//////////////////////////////////////////////////

type QuitRequest struct {
  Request
}

func (rq *QuitRequest) Handle(dispatcher *Dispatcher) (Respondable, error) {
  dispatcher.ClientQuit(rq.client)
  rs := NewQuitResponse()
  return &rs, nil
}

//////////////////////////////////////////////////

var Requests = map[string] func() Requestable {
  "CHAT"  : func() Requestable { return new(Request) },
  "USER"  : func() Requestable { return new(UserRequest) },
  "USERS" : func() Requestable { return new(UsersRequest) },
  "ROOMS" : func() Requestable { return new(RoomsRequest) },
  "JOIN"  : func() Requestable { return new(JoinRequest) },
  "PART"  : func() Requestable { return new(PartRequest) },
  "SAY"   : func() Requestable { return new(SayRequest) },
  "QUIT"  : func() Requestable { return new(QuitRequest) },
}


