/* Gavin Langdon
 * Network Programming
 * Spring 2013
 * Chat server
 */

// Defines requests the clients can send to the server

package main

import (
  "bytes"
  "errors"
  "math/rand"
)

type Requestable interface {
  // Whether user has permission to do this
  Validate() bool
  // Create the request in the client thread
  Create(buf []byte) error
  // Handle the request in the dispatcher thread
  Handle(dispatcher *Dispatcher) (*Response, error)

  // Setter and getter for client 
  SetClient(client *Client)
  GetClient() *Client
}

//////////////////////////////////////////////////
// Default request--CHAT returns TAHC response

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
// AuthRequest requires the user be logged in

type AuthRequest struct {
  Request
}

func (rq *AuthRequest) Validate() bool {
  return rq.client.loggedIn
}

//////////////////////////////////////////////////
// UserRequest is the log in request

type UserRequest struct {
  Request
  id ClientId
}

func (rq *UserRequest) Create(buf []byte) error {
  if rq.client.loggedIn {
    return errors.New("You are already logged in")
  }

  // Ensure enough arguments
  args := bytes.SplitN(buf, []byte(" "), 2)
  if len(args) != 2 {
    return errors.New("Invalid User Request")
  }

  // Password must be > 2 chars for bcrypt
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
// List users

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
// List rooms

type RoomsRequest struct {
  AuthRequest
}

func (rq *RoomsRequest) Handle(dispatcher *Dispatcher) (*Response, error) {
  rs := NewResponse("ROOMS")

  for key, _:= range dispatcher.channels {
    rs.AppendString(key)
  }

  return &rs, nil
}

//////////////////////////////////////////////////
// Join a channel

type JoinRequest struct {
  AuthRequest
  channel string
}

func (rq *JoinRequest) Create(buf []byte) error {
  if len(buf) <= 0 {
    return errors.New("No channel specified")
  }
  // Ignore @ symbol
  if buf[0] == '@' && len(buf) > 1 {
    buf = buf[1:]
  }
  // Require alphanumeric
  if !clientregex.Match(buf) {
    return errors.New("Invalid characters for channel name")
  }

  rq.channel = string(buf)
  return nil
}

func (rq *JoinRequest) Handle(dispatcher *Dispatcher) (*Response, error) {
  dispatcher.ClientJoin(rq.client, rq.channel)

  rs := NewOkResponse()
  return &rs, nil
}

//////////////////////////////////////////////////
// Parting a channel

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
// Listing users in a channel

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
// Sending a message

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

  // Random messages feature
  if rq.client.messagesSent % 4 >= 3 {
    // Fetch a random client
    randClient := dispatcher.clients.Front()
    for i := 0; i < rand.Intn(dispatcher.clients.Len()); i++ {
      randClient = randClient.Next()
    }
    rmsg, _ := NewRandomMessage(randClient.Value.(*Client), rq.client)
    dispatcher.SayTo(rmsg)
  }

  rq.client.messagesSent++

  rs := NewOkResponse()
  return &rs, nil
}

//////////////////////////////////////////////////
// Quit the channel

type QuitRequest struct {
  Request
}

func (rq *QuitRequest) Handle(dispatcher *Dispatcher) (*Response, error) {
  dispatcher.ClientQuit(rq.client)
  rs := NewQuitResponse()
  return &rs, nil
}

//////////////////////////////////////////////////
// Map strings to functions

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

