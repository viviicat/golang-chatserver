package main

import (
  "net"
  "time"
)


type FauxConn struct {
  inCh chan []byte
  addr net.Addr
  mainConn net.PacketConn
}

func (c *FauxConn) Read(b []byte) (int, error) {
  return copy(b, <-c.inCh), nil
}

func (c *FauxConn) Write(b []byte) (int, error) {
  return c.mainConn.WriteTo(b, c.addr)
}

func (c *FauxConn) Close() error {
  close(c.inCh)
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

func NewFauxConn(addr net.Addr, mainConn net.PacketConn) *FauxConn {
  return &FauxConn{make(chan []byte, 10), addr, mainConn}
}

