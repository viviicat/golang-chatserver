/* Gavin Langdon
 * Network Programming
 * Spring 2013
 * Chat server
 */


package main

import (
  "net"
  "container/list"
  "flag"
)

var VerboseMode = flag.Bool("v", false, "Verbose mode--enables logging of messages")
var ListenPort string

func init() {
  flag.Parse()

  if flag.NArg() < 1 {
    panic("Port not specified")
  }

  ListenPort = ":" + flag.Arg(0)
}

// Custom list so we have a Find method
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

// Listen to the TCP connection
func listen(listener net.Listener, mainChan chan net.Conn) {
  for {
    conn, err := listener.Accept()
    if err != nil {
      err.Error()
      return
    }
    // Send new connection to the dispatcher loop
    mainChan <- conn
  }
}

type UDPListener struct {
  connSet map[string] *FauxConn
  mainConn net.PacketConn
  closeCh chan string
}

func (l *UDPListener) flushCloses() {
  for {
    select {
      // Since the deletion does not occur until after the ReadFrom returns,
      // the user is technically not deleted until the next udp message is received.
      // This means that if the same user were to reconnect in the next udp message,
      // he would be deleted without a response. However udp doesn't guarantee messages
      // be received at all, so the user can just be forced to retype the message.
    case closedIP := <-l.closeCh:
      delete(l.connSet, closedIP)

    default:
      return
    }
  }
}

// Listen to the TCP connection
func listenUDP(mainChan chan net.Conn) error {
  conn, err := net.ListenPacket("udp", ListenPort)
  if err != nil {
    return err
  }

  l := UDPListener{make(map[string] *FauxConn), conn, make(chan string, 10)}

  for {
    l.flushCloses()

    buf := make([]byte, 1024)
    count, addr, err := conn.ReadFrom(buf)
    if err != nil {
      return err
    }

    fc := l.connSet[addr.String()]
    if fc == nil {
      fc = NewFauxConn(addr, l.mainConn, l.closeCh)
      l.connSet[addr.String()] = fc
      // Inform the dispatcher of the new connection
      mainChan <- fc
    }
    // Send buffer to the client's buffer channel
    fc.inCh <- buf[:count]
  }

  return nil
}

func main() {
  var err error

  mainChan := make(chan net.Conn, 10)

  // Start dispatch loop
  go Dispatch(mainChan)

  listener, err := net.Listen("tcp", ListenPort)
  defer listener.Close()
  if err != nil {
    return
  }

  // start udp loop
  go listenUDP(mainChan)
  // Start TCP loop
  listen(listener, mainChan)

}
