package main

import (
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


func listenUDP(mainChan chan net.Conn) error {
  conn, err := net.ListenPacket("udp", ":12180")
  if err != nil {
    return err
  }

  udpConnSet := make(map[string] *FauxConn)

  for {
    buf := make([]byte, 1024)
    count, addr, err := conn.ReadFrom(buf)
    if err != nil {
      return err
    }

    fc := udpConnSet[addr.String()]
    if fc == nil {
      fc = NewFauxConn(addr, conn)
      udpConnSet[addr.String()] = fc
      mainChan <- fc
    }
    fc.inCh <- buf[:count]

  }

  return nil
}

func main() {
  var err error

  mainChan := make(chan net.Conn, 10)

  go Dispatch(mainChan)

  listener, err := net.Listen("tcp", ":12180")
  defer listener.Close()
  if err != nil {
    return
  }

  go listenUDP(mainChan)
  listen(listener, mainChan)

}
