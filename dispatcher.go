package main

import (
  "net"
  "container/list"
  "errors"
  "code.google.com/p/go.crypto/bcrypt"
)

func (d *Dispatcher) NewClient(conn net.Conn) Client {
  var cl Client
  cl.responseCh = make(chan *Response, 10)
  cl.requestCh = d.requestCh
  cl.loggedIn = false
  cl.loginTries = 0
  cl.Conn = conn

  return cl
}


type Dispatcher struct {
  // List of clients
  clients *List

  // Set of unique usernames mapping to password and login status
  clientSet map[string] ClientInfo

  // Map of channels (each channel is a list of clients)
  channels map[string] *List

  // The channel the clients will send requests to
  requestCh chan Requestable

  // The channel the main thread will send new connections to
  connCh chan net.Conn
}

func NewDispatcher(connCh chan net.Conn) Dispatcher {
  disp := Dispatcher{
    &List{list.New()},
    make(map[string] ClientInfo),
    make(map[string] *List),
    make(chan Requestable, 10),
    connCh}

  return disp
}

func (d *Dispatcher) ClientLogin(client *Client, username string, password []byte) error {
  // If user is already registered
  if _, hit := d.clientSet[username]; hit {
    // User logged in already
    if d.clientSet[username].loggedIn {
      return errors.New("This username is already in the channel")
    }

    // Password incorrect
    if bcrypt.CompareHashAndPassword(d.clientSet[username].password, password) != nil {
      client.loginTries++
      if client.loginTries >= 3 {
        err := NewDisconnectError("Max login tries. Bye")
        return &err
      }

      return errors.New("Invalid password specified for user")
    }

  }

  crypt, err := bcrypt.GenerateFromPassword(password, 0)
  if err != nil {
    return err
  }

  client.username = username

  client.loggedIn = true
  client.loginTries = 0

  d.clientSet[username] = ClientInfo{crypt, true, client}

  return nil
}

func (d *Dispatcher) GetClient(username string) (*Client, error) {
  if info, ok := d.clientSet[username]; ok {
    return info.client, nil
  }
  return nil, errors.New("Client not found")
}

func (d *Dispatcher) GetChannel(channel string) (*List, error) {
  if channelList := d.channels[channel]; channelList != nil {
    return channelList, nil
  }
  return nil, errors.New("Channel does not exist")
}

func (d *Dispatcher) ClientJoin(client *Client, channel string) error {
  channelList, _ := d.GetChannel(channel)

  if channelList == nil {
    channelList = &List{list.New()}
    d.channels[channel] = channelList
  }
  if e := channelList.Find(client); e == nil {
    channelList.PushBack(client)
  }

  return nil
}

func (d *Dispatcher) ClientPartAll(client *Client) {
  for key, _ := range d.channels {
    d.ClientPart(client, key)
  }
}

func (d *Dispatcher) ClientPart(client *Client, channel string) error {
  channelList, err := d.GetChannel(channel)
  if err != nil {
    return err
  }

  if e := channelList.Find(client); e != nil {
    channelList.Remove(e)
    return nil
  }

  return errors.New("You are not in this channel")
}

func (d *Dispatcher) SayTo(message *Message) error {

  if message.target[0] == '@' {
    channelList, err := d.GetChannel(message.target[1:])
    if err != nil {
      return err
    }

    for e := channelList.Front(); e != nil; e = e.Next() {
      message.WriteTo(e.Value.(*Client))
    }
    return nil
  }

  client, err := d.GetClient(message.target)
  if err != nil {
    return err
  }

  message.WriteTo(client)

  return nil
}

func (d *Dispatcher) ClientQuit(client *Client) {
  d.ClientPartAll(client)

  if e := d.clients.Find(client); e != nil {
    d.clients.Remove(e)
  }

  cs := d.clientSet[client.username]
  cs.loggedIn = false
  d.clientSet[client.username] = cs
}


func Dispatch(connCh chan net.Conn) {
  dispatcher := NewDispatcher(connCh)

  for {
    select {
      case conn := <-dispatcher.connCh:
        cl := dispatcher.NewClient(conn)
        dispatcher.clients.PushBack(&cl)

        go cl.Serve()

      case request := <-dispatcher.requestCh:
        response, err := request.Handle(&dispatcher)
        if err != nil {
          if response == nil {
            er := NewErrorResponse(err)
            response = &er
          } else if response.Quit {
            dispatcher.ClientQuit(request.GetClient())
          }
        }
        request.GetClient().responseCh <- response
    }
  }
}


