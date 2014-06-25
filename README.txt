Gavin Langdon
Network Programming
Spring 2013
Chat Server

This server is written in Golang, which can be installed in Ubuntu with:

  sudo apt-get install golang


To quickly compile and run you can use the command `go run' as follows:

  go run *.go -v 12180


You can talk to the server with netcat. There is no client (yet).

$ nc localhost 12180
USER me password
> OK
JOIN random_channel
> OK
SAY otheruser 17 this is a message
> OK
SAY @random_channel 15 hi guys!!
> FROM me 15 hi guys!!
> OK


etc.
