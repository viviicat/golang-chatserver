/* Gavin Langdon
 * Network Programming
 * Spring 2013
 * Chat server
 */

// Specifies messages sent by clients to clients

package main

import (
  "strconv"
  "fmt"
  "errors"
  "bytes"
  "math/rand"
)

type Message struct {
  from string
  target string
  chunks [][]byte
}

// Random messages to send per the requirements
var RandomStrings = []string {
  "It’s a hug, Michael. I’m hugging you.",
  "I think you’re going to be surprised at some of your phrasing.",
  "Not tricks, Michael, illusions. A trick is something a whore does for money.",
  "I’m a failure. I can’t even fake the death of a stripper.",
  "There’s so many poorly chosen words in that sentence.",
  "She’s not that Mexican, Mom, she’s my Mexican. And she’s Colombian or something.",
  "I’ve opened a door here that I regret.",
  "I hear the jury’s still out on science.",
  "Army had half a day.",
  "Only two of those words describe Mom, so I know you’re lying to me.",
  "I don’t understand the question, and I won’t respond to it.",
}

// Find a random message and send to client
func NewRandomMessage(from *Client, to *Client) (*Message, error) {
  str := RandomStrings[rand.Intn(len(RandomStrings)-1)]
  var msg Message
  msg.from = from.username
  msg.target = to.username

  data := []byte(strconv.Itoa(len(str)) + " " + str)
  _, err := msg.FirstMessageChunk(data)
  return &msg, err
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
  more, err := msg.FirstMessageChunk(spl[1])
  if err != nil {
    return nil, err
  }

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

func (m *Message) FirstMessageChunk(data []byte) (bool, error) {
  // If the message is chunked, use this opportunity to wait for all the chunks
  // (we're still in the client goroutine)
  more, err := m.AddMessageChunk(data)
  if err != nil {
    return false, err
  }

  // Add FROM header to first chunk
  m.chunks[0] = append([]byte("FROM " + m.from + " "), m.chunks[0]...)
  return more, nil
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

// Write all chunks in sequence to the client
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

