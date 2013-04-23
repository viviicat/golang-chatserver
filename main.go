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

type UDPListener struct {
  connSet map[string] *FauxConn
  mainConn net.PacketConn
}


func listenUDP(mainChan chan net.Conn) error {
  conn, err := net.ListenPacket("udp", ":12180")
  if err != nil {
    return err
  }

  l := UDPListener{make(map[string] *FauxConn), conn}

  for {
    buf := make([]byte, 1024)
    count, addr, err := conn.ReadFrom(buf)
    if err != nil {
      return err
    }

    fc := l.connSet[addr.String()]
    if fc == nil {
      fc = NewFauxConn(addr, &l)
      l.connSet[addr.String()] = fc
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
