package main

import (
  "strconv"
  "io"
  "fmt"
  "errors"
  "bytes"
)

type Message struct {
  from string
  target string
  data []byte
  chunks *[][]byte
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

  for more {
    count, err := client.Read(data)
    if err != nil {
      return nil, err
    }
    if count <= 0 {
      continue
    }

    more, err = msg.AddMessageChunk(bytes.Trim(data[:count], "\r\n"))
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
      // If this is not C0 append
      if count > 0 {
        m.data = append(m.data, spl[1]...)
      }
      // Return false if C0/done
      return count != 0, nil
    }
  } else if _, err := strconv.Atoi(string(spl[0])); len(spl) == 2 && err == nil {
    m.data = append(m.data, spl[1]...)
    // Single packet, return false
    return false, nil
  }
  return false, errors.New("Malformed packet " + string(data))
}

// Format a message to be chunked
func GetMessageChunk(dataRemaining []byte) ([]byte, []byte) {
  if len(dataRemaining) <= 0 {
    return []byte("C0\r\n"), nil
  }

  chunkLen := len(dataRemaining)
  if chunkLen > 999 {
    chunkLen = 999
  }
  chunk := []byte("C" + strconv.Itoa(chunkLen) + " ")
  return append(append(chunk, dataRemaining[:chunkLen]...), []byte("\r\n")...), dataRemaining[chunkLen:]
}

func (m *Message) GetChunks() *[][]byte {
  if m.chunks != nil {
    return m.chunks
  }

  header := "FROM " + m.from + " "

  // If message shorter than 99 then just send it with length prefixed
  if len(m.data) <= 99 {
    fmt.Println(len(m.data))
    header += strconv.Itoa(len(m.data)) + " "
    return &[][]byte{append(append([]byte(header), m.data...), []byte("\r\n")...)}
  }

  // Otherwise, the first chunk should include the header
  chunk, next := GetMessageChunk(m.data)
  arr := [][]byte{append([]byte(header), chunk...)}

  for next != nil {
    chunk, next = GetMessageChunk(next)
    arr = append(arr, chunk)
  }

  m.chunks = &arr
  return &arr
}

func (m *Message) WriteTo(w io.Writer) (n int, err error) {
  ch := m.GetChunks()
  n = 0
  for i := range *ch {
    wr, err := w.Write((*ch)[i])
    if err != nil {
      return -1, err
    }
    n += wr
  }
  return n, nil
}

