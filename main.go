package main

import (
  "fmt"
  "net"
  "container/list"
  "errors"
  "code.google.com/p/go.crypto/bcrypt"
)

type List struct {
  *list.List
}

func (l *List) Find(value interface{}) *list.Element {
  for e := l.Front(); e != nil; e = e.Next() {
    if e.Value == value {
      return e
    }
  }
  return nil
}

func (d *Dispatcher) NewClient(conn net.Conn) Client {
  var cl Client
  cl.responseCh = make(chan Respondable, 10)
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

  client.username = username

  client.loggedIn = true
  client.loginTries = 0

  crypt, err := bcrypt.GenerateFromPassword(password, 0)
  if err != nil {
    return err
  }

  d.clientSet[username] = ClientInfo{crypt, true}

  return nil
}

func (d *Dispatcher) ClientJoin(client *Client, channel string) {
  channelList := d.channels[channel]
  if channelList == nil {
    channelList = &List{list.New()}
    d.channels[channel] = channelList
    fmt.Println("Created new channel", channel)
  }
  if e := channelList.Find(client); e == nil {
    fmt.Println("Added user to channel", channel)
    channelList.PushBack(client)
  }
}

func (d *Dispatcher) ClientPartAll(client *Client) {
  for key, _ := range d.channels {
    d.ClientPart(client, key)
  }
}

func (d *Dispatcher) ClientPart(client *Client, channel string) {
  if channelList := d.channels[channel]; channelList != nil {
    if e := channelList.Find(client); e != nil {
      fmt.Println("Removed user from channel", channel)
      channelList.Remove(e)
    }
  }
}

func (d *Dispatcher) ClientSayTo(client *Client, message *Message) {
  if channelList := d.channels[message.target]; channelList != nil {
    for e := channelList.Front(); e != nil; e = e.Next() {
      response := NewMessageResponse(message)
      response.Send(e.Value.(net.Conn))
    }
  }
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
        fmt.Println("Caught a client. serving!")

        cl := dispatcher.NewClient(conn)
        dispatcher.clients.PushBack(&cl)

        go cl.Serve()

      case request := <-dispatcher.requestCh:
        response, err := request.Handle(&dispatcher)
        if err != nil {
          fmt.Println(err)
          if response == nil {
            er := NewErrorResponse(err)
            response = &er
          } else if _, ok := response.(*QuitResponse); ok {
            dispatcher.ClientQuit(request.GetClient())
          }
        }
        request.GetClient().responseCh <- response
    }
  }
}

func listen(listener net.Listener, mainChan chan net.Conn) {
  for {
    conn, err := listener.Accept()
    if err != nil {
      err.Error()
      return
    }
    mainChan <- conn
  }
}

func main() {
  fmt.Println("Initializing Chat Server...")
  var err error

  mainChan := make(chan net.Conn, 10)

  go Dispatch(mainChan)

  listener, err := net.Listen("tcp", ":12180")
  if err != nil {
    fmt.Println(err)
    return
  }

  /*laddr, err := net.ResolveIPAddr("ip", ":12180")
  if err != nil {
    fmt.Println(err)
    return
  }

  conn, err := net.ListenIP("ip",laddr)
  if err != nil {
    fmt.Println(err)
    return
  }*/

  listen(listener, mainChan)

}
