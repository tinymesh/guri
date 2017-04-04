package main

import (
	"errors"
	"io"
	"log"
	"time"
)

type StdioRemote struct {
	reader  io.Reader
	writer  io.Writer
	channel chan []byte
}

func ConnectStdio(input io.Reader, output io.Writer) (*StdioRemote, error) {
	log.Printf("stdio:open uri=-\n")
	return &StdioRemote{
		reader:  input,
		writer:  output,
		channel: make(chan []byte, 256),
	}, nil
}

func (remote *StdioRemote) Channel() chan []byte {
	return remote.channel
}

func (remote *StdioRemote) Close() error {
	return nil
}

func (remote *StdioRemote) Open() chan []byte {

	go func() {
		for {
			buf, err := remote.Recv(-1)

			if nil != err {
				log.Fatalf("error[stdio:recv] %v\n", err)
			} else if 0 == len(buf) {
				log.Fatalf("error[stdio:recv] %v\n", errors.New("EOF"))
			}

			remote.channel <- buf
		}
	}()

	return remote.channel
}

func (remote *StdioRemote) Recv(timeout time.Duration) ([]byte, error) {
	buf := make([]byte, 256)
	n, err := io.ReadAtLeast(remote.reader, buf, 1)

	if nil != err {
		return nil, err
	}

	log.Printf("stdio:recv %v\n", buf[:n])
	return buf[:n], nil
}

func (remote *StdioRemote) Write(buf []byte, timeout time.Duration) (int, error) {
	log.Printf("stdio:write %v\n", buf)
	return remote.writer.Write(buf)
}
