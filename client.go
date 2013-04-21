package main

import (
  "fmt"
  "net"
  "bytes"
  "regexp"
  "errors"
)

type ClientId struct {
  username string
  password []byte
}

type ClientInfo struct {
  password []byte
  loggedIn bool
}

var clientregex = regexp.MustCompile("^[[:alnum:]]+$")

func NewClientId(username string, password []byte) (ClientId, error){
  if !clientregex.MatchString(username) {
    return ClientId{}, errors.New("Invalid username chars provided")
  }

  return ClientId{username, password}, nil
}


type Client struct {
  net.Conn

  responseCh chan Respondable
  requestCh chan Requestable

  username string

  loggedIn bool
  loginTries int
}

func (cl *Client) HandleRequest(request []byte) (bool, error) {
  temp := bytes.SplitN(request, []byte(" "), 2)
  code := temp[0]
  var data []byte
  if len(temp) >= 2 {
    data = temp[1]
  }

  for request_str := range Requests {
    if bytes.Compare(bytes.ToUpper(code), []byte(request_str)) == 0 {
      rq := Requests[request_str]()
      rq.SetClient(cl)

      err := rq.Create(data)
      if err != nil {
        return false, err
      }
      cl.requestCh <-rq
      response := <-cl.responseCh
      response.Send(cl.Conn)

      if _, ok := response.(*QuitResponse); ok {
        return true, nil
      }
      return false, nil
    }
  }

  return false, errors.New("Invalid request code")
}

func (cl *Client) Close() error {
  // Defer connection closing
  defer cl.Conn.Close()
 
  // Create a dummy quit request and shuttle it to the other goroutine
  // so its data is cleaned up
  var qr QuitRequest
  qr.SetClient(cl)
  cl.requestCh <-&qr
  return nil
}

func (cl *Client) Serve() {
  defer cl.Close()

  buf := make([]byte, 128)

  for {
    count, err := cl.Read(buf)
    if err != nil {
      fmt.Println(err)
        return
    }
    if count <= 0 {
      continue
    }

    request := bytes.Trim(buf[:count], "\r\n")
    disconn, err := cl.HandleRequest(request)

    if err != nil {
      fmt.Println(err)
      response := NewErrorResponse(err)
      response.Send(cl.Conn)

      if _, ok := err.(*DisconnectError); ok {
        fmt.Println("Got disconnect error. Disconnecting")
        return
      }
    }

    if disconn {
      return
    }
  }
}

