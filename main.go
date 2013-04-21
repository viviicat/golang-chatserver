package main

import (
  "fmt"
  "net"
  "container/list"
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
