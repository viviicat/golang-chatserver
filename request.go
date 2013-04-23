package main

import (
  "bytes"
  "errors"
)

type Requestable interface {
  Validate() bool
  Create(buf []byte) error
  Handle(dispatcher *Dispatcher) (*Response, error)
  SetClient(client *Client)
  GetClient() *Client
}

//////////////////////////////////////////////////

type Request struct {
  client *Client
}


func (rq *Request) Validate() bool {
  return true
}

func (rq *Request) Create(buf []byte) error {
  return nil
}

func (rq *Request) Handle(dispatcher *Dispatcher) (*Response, error) {
  rs := NewResponse("TAHC")
  return &rs, nil
}

func (rq *Request) SetClient(client *Client) {
  rq.client = client
}

func (rq *Request) GetClient() *Client {
  return rq.client
}

//////////////////////////////////////////////////

type AuthRequest struct {
  Request
}

func (rq *AuthRequest) Validate() bool {
  return rq.client.loggedIn
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

  if len(args[1]) < 3 {
    return errors.New("Password is too short")
  }

  var err error
  rq.id, err = NewClientId(string(args[0]), args[1])
  if err != nil {
    return err
  }

  return nil
}

func (rq *UserRequest) Handle(dispatcher *Dispatcher) (*Response, error) {
  if err := dispatcher.ClientLogin(rq.client, rq.id.username, rq.id.password); err != nil {
    var rs *Response
    if _, ok := err.(*DisconnectError); ok {
      dr := NewFatalErrorResponse(err)
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
  AuthRequest
}

func (rq *UsersRequest) Handle(dispatcher *Dispatcher) (*Response, error) {
  rs := NewResponse("USERS")

  for e := dispatcher.clients.Front(); e != nil; e = e.Next() {
    rs.AppendString(e.Value.(*Client).username)
  }

  return &rs, nil
}

//////////////////////////////////////////////////

type RoomsRequest struct {
  AuthRequest
}

func (rq *RoomsRequest) Handle(dispatcher *Dispatcher) (*Response, error) {
  rs := NewResponse("ROOMS")

  for key, _ := range dispatcher.channels {
    rs.AppendString(key)
  }

  return &rs, nil
}

//////////////////////////////////////////////////

type JoinRequest struct {
  AuthRequest
  channel string
}

func (rq *JoinRequest) Create(buf []byte) error {
  if len(buf) <= 0 {
    return errors.New("No channel specified")
  }
  if buf[0] == '@' && len(buf) > 1 {
    rq.channel = string(buf[1:])
  } else {
    rq.channel = string(buf)
  }
  return nil
}

func (rq *JoinRequest) Handle(dispatcher *Dispatcher) (*Response, error) {
  dispatcher.ClientJoin(rq.client, rq.channel)

  rs := NewOkResponse()
  return &rs, nil
}

//////////////////////////////////////////////////

type PartRequest struct {
  JoinRequest
}

func (rq *PartRequest) Handle(dispatcher *Dispatcher) (*Response, error) {
  err := dispatcher.ClientPart(rq.client, rq.channel)
  if err != nil {
    return nil, err
  }

  rs := NewOkResponse()
  return &rs, nil
}

//////////////////////////////////////////////////

type ListRequest struct {
  JoinRequest
}

func (rq *ListRequest) Handle(dispatcher *Dispatcher) (*Response, error) {

  if list := dispatcher.channels[rq.channel]; list != nil {
    rs := NewResponse("LIST")
    for e := list.Front(); e != nil; e = e.Next() {
      rs.AppendString(e.Value.(*Client).username)
    }
    return &rs, nil
  }

  return nil, errors.New("Channel does not exist")
}

//////////////////////////////////////////////////


type SayRequest struct {
  AuthRequest
  Message
}

func (rq *SayRequest) Create(buf []byte) error {
  msg, err := NewMessage(buf, rq.client)
  if err != nil {
    return err
  }
  rq.Message = *msg
  return nil
}

func (rq *SayRequest) Handle(dispatcher *Dispatcher) (*Response, error) {
  err := dispatcher.SayTo(&rq.Message)
  if err != nil {
    return nil, err
  }

  rs := NewOkResponse()
  return &rs, nil
}

//////////////////////////////////////////////////

type QuitRequest struct {
  Request
}

func (rq *QuitRequest) Handle(dispatcher *Dispatcher) (*Response, error) {
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
  "LIST"  : func() Requestable { return new(ListRequest) },
  "SAY"   : func() Requestable { return new(SayRequest) },
  "QUIT"  : func() Requestable { return new(QuitRequest) },
}


