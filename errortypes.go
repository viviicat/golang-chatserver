/* Gavin Langdon
 * Network Programming
 * Spring 2013
 * Chat server
 */

package main

type DisconnectError struct {
  err string
}

func NewDisconnectError(err string) DisconnectError {
  return DisconnectError{err}
}

func (e *DisconnectError) Error() string {
  return e.err
}
