package main

import (
  "net"
  "time"
)


type FauxConn struct {
  inCh chan []byte
  addr net.Addr
  listener *UDPListener
}

func (c *FauxConn) Read(b []byte) (int, error) {
  return copy(b, <-c.inCh), nil
}

func (c *FauxConn) Write(b []byte) (int, error) {
  return c.listener.mainConn.WriteTo(b, c.addr)
}

func (c *FauxConn) Close() error {
  close(c.inCh)
  c.listener.connSet[c.addr.String()] = nil
  return nil
}

func (c *FauxConn) LocalAddr() net.Addr {
  return c.addr
}

func (c *FauxConn) RemoteAddr() net.Addr {
  return c.addr
}

func (c *FauxConn) SetDeadline(t time.Time) error {
  return nil
}

func (c *FauxConn) SetReadDeadline(t time.Time) error {
  return nil
}

func (c *FauxConn) SetWriteDeadline(t time.Time) error {
  return nil
}

func NewFauxConn(addr net.Addr, listener *UDPListener) *FauxConn {
  return &FauxConn{make(chan []byte, 10), addr, listener}
}

