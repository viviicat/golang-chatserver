/* Gavin Langdon
 * Network Programming
 * Spring 2013
 * Chat server
 */

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
  client *Client
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

  responseCh chan *Response
  requestCh chan Requestable

  username string

  loggedIn bool
  loginTries int

  messagesSent int
}

func (cl *Client) String() string {
  str := cl.RemoteAddr().String()
  if cl.loggedIn {
    return cl.username + " (" + str + ")"
  }
  return str
}

func (cl *Client) HandleRequest(request []byte) (bool, error) {
  if *VerboseMode {
    fmt.Println("RCVD from", cl.String()+":", string(request))
  }

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

      if !rq.Validate() {
        return false, errors.New("Not authorized to do this")
      }

      err := rq.Create(data)
      if err != nil {
        return false, err
      }
      cl.requestCh <-rq
      response := <-cl.responseCh
      response.WriteTo(cl)

      return response.Quit, nil
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

  buf := make([]byte, 1024)

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
      response := NewErrorResponse(err)
      response.WriteTo(cl)

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

