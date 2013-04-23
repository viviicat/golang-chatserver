/* Gavin Langdon
 * Network Programming
 * Spring 2013
 * Chat server
 */


// A structure that acts like a net.Conn but actually gets its data from a master connection,
// so that we can do UDP but treat separate ip/port combos as separate connections

package main

import (
  "net"
  "time"
)


type FauxConn struct {
  inCh chan []byte
  addr net.Addr
  mainConn net.PacketConn

  closeCh chan string
}

// Read from the channel buffer
func (c *FauxConn) Read(b []byte) (int, error) {
  return copy(b, <-c.inCh), nil
}

// Write immediately with the main connection
func (c *FauxConn) Write(b []byte) (int, error) {
  return c.mainConn.WriteTo(b, c.addr)
}

// Remove listener reference to this when we close
func (c *FauxConn) Close() error {
  c.closeCh <- c.addr.String()
  return nil
}


func (c *FauxConn) RemoteAddr() net.Addr {
  return c.addr
}

// Dummies to satisfy the interface

func (c *FauxConn) LocalAddr() net.Addr {
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

func NewFauxConn(addr net.Addr, mainConn net.PacketConn, closeCh chan string) *FauxConn {
  return &FauxConn{make(chan []byte, 10), addr, mainConn, closeCh}
}

