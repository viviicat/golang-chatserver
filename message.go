package main

import (
  "strconv"
  "fmt"
  "errors"
  "bytes"
)

type Message struct {
  from string
  target string
  chunks [][]byte
}


// Creates a new message from a request data packet and returns it
func NewMessage(data []byte, client *Client) (*Message, error) {
  spl := bytes.SplitN(data, []byte(" "), 2)

  if len(spl) < 2 {
    return nil, errors.New("Missing argument(s)")
  }

  var msg Message

  msg.from = client.username
  msg.target = string(spl[0])

  // If the message is chunked, use this opportunity to wait for all the chunks
  // (we're still in the client goroutine)
  more, err := msg.AddMessageChunk(spl[1])
  if err != nil {
    return nil, err
  }

  // Add FROM header to first chunk
  msg.chunks[0] = append([]byte("FROM " + msg.from + " "), msg.chunks[0]...)

  for more {
    buf := make([]byte, 1024)
    count, err := client.Read(buf)
    if err != nil {
      return nil, err
    }
    if count <= 0 {
      continue
    }

    if *VerboseMode {
      fmt.Println(string(buf[:count]))
    }

    more, err = msg.AddMessageChunk(bytes.Trim(buf[:count], "\r\n"))
    if err != nil {
      return nil, err
    }
  }

  return &msg, nil
}

func (m *Message) AddMessageChunk(data []byte) (bool, error) {
  // Split message length specifier and message
  spl := bytes.SplitN(data, []byte(" "), 2)

  // If the beginning of the length specifier is C we're chunked
  if spl[0][0] == 'C' {
    // Make sure this is an actual length specifier and get the count
    if count, err := strconv.Atoi(string(spl[0][1:])); err == nil {
      if count > 999 {
        return false, errors.New("Packet size too large")
      }
      m.chunks = append(m.chunks, append(data, '\n'))
      // Return false if C0/done
      return count != 0, nil
    }
  } else if count, err := strconv.Atoi(string(spl[0])); len(spl) == 2 && err == nil {
    if count > 99 {
      return false, errors.New("Packet size too large for short format")
    }
    m.chunks = append(m.chunks, append(data, '\n'))
    // Single packet, return false
    return false, nil
  }
  return false, errors.New("Malformed packet ")
}


func (m *Message) GetChunks() [][]byte {
  return m.chunks
}

func (m *Message) WriteTo(c *Client) (n int, err error) {
  var strbuf bytes.Buffer

  if *VerboseMode {
    strbuf.WriteString("SENT to " + c.String() + ": ")
  }

  ch := m.GetChunks()
  n = 0
  for i := range ch {
    wr, err := c.Write(ch[i])
    if *VerboseMode {
      strbuf.Write(ch[i])
    }
    if err != nil {
      return -1, err
    }
    n += wr
  }
  if *VerboseMode {
    fmt.Println(strbuf.String())
  }
  return n, nil
}

