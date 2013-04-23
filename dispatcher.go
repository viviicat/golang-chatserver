/* Gavin Langdon
 * Network Programming
 * Spring 2013
 * Chat server
 */

// The dispatcher gets requests from the clients and handles them in a single thread,
// dispatching responses and messages

package main

import (
  "net"
  "container/list"
  "errors"
  // This is working but it requires a go get
  //"code.google.com/p/go.crypto/bcrypt"
  "bytes"
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
    //if bcrypt.CompareHashAndPassword(d.clientSet[username].password, password) != nil {
    if bytes.Compare(d.clientSet[username].password, password) != 0 {
      client.loginTries++
      if client.loginTries >= 3 {
        err := NewDisconnectError("Max login tries. Bye")
        return &err
      }

      return errors.New("Invalid password specified for user")
    }

  }

  /*crypt, err := bcrypt.GenerateFromPassword(password, 0)
  if err != nil {
    return err
  }*/

  client.username = username

  client.loggedIn = true
  client.loginTries = 0

  d.clientSet[username] = ClientInfo{password, true, client}

  return nil
}

// Fetch a client by username
func (d *Dispatcher) GetClient(username string) (*Client, error) {
  // Find client
  if info, ok := d.clientSet[username]; ok {
    // Ensure client is logged in
    if info.client != nil {
      return info.client, nil
    }
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
  // Find the specified channel
  channelList, _ := d.GetChannel(channel)

  // If it doesn't exist, make a new one
  if channelList == nil {
    channelList = &List{list.New()}
    d.channels[channel] = channelList
  }
  // If the client is not in the channel, add them
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
  // Find channel
  channelList, err := d.GetChannel(channel)
  if err != nil {
    return err
  }

  // If user is in channel, remove
  if e := channelList.Find(client); e != nil {
    channelList.Remove(e)
    return nil
  }

  return errors.New("You are not in this channel")
}

// Send a message
func (d *Dispatcher) SayTo(message *Message) error {
  // Messages starting with @ will be to channels
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

  // Otherwise send a single message to a client
  client, err := d.GetClient(message.target)
  if err != nil {
    return err
  }

  message.WriteTo(client)

  return nil
}

func (d *Dispatcher) ClientQuit(client *Client) {
  // leave all channels
  d.ClientPartAll(client)

  // Remove from client list
  if e := d.clients.Find(client); e != nil {
    d.clients.Remove(e)
  }

  // Set state in saved client list
  cs := d.clientSet[client.username]
  cs.loggedIn = false
  // Remove reference to this client instance
  cs.client = nil
  d.clientSet[client.username] = cs
}


// Dispatch loop adds new connections and fetches requests from existing
// connections
func Dispatch(connCh chan net.Conn) {
  dispatcher := NewDispatcher(connCh)

  for {
    select {
      // New connection
      case conn := <-dispatcher.connCh:
        cl := dispatcher.NewClient(conn)
        dispatcher.clients.PushBack(&cl)

        go cl.Serve()

      // Existing connection request
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
        // send client loop the response
        request.GetClient().responseCh <- response
    }
  }
}


