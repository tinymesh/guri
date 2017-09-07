package guri

import (
	"errors"
	"io"
	"log"
	"time"
)

// StdioRemote stdio endpoint
type StdioRemote struct {
	reader  io.Reader
	writer  io.Writer
	channel chan []byte
	done    chan interface{}
}

// ConnectStdio create a new stdio connection
func ConnectStdio(input io.Reader, output io.Writer) (*StdioRemote, error) {
	log.Printf("stdio:open uri=-\n")

	remote := &StdioRemote{
		reader: input,
		writer: output,
		done:   make(chan interface{}, 2),
	}

	remote.Connect()

	return remote, nil
}

// Connect dial into stdio remote
func (remote *StdioRemote) Connect() error {
	data := make(chan []byte, 256)

	go func() {
		for {
			buf := make([]byte, 256)
			n, err := io.ReadAtLeast(remote.reader, buf, 1)

			if nil != err {
				log.Printf("stdio:err - failed to read data %v\n", err)
				data <- []byte("")
				return
			}

			select {
			case data <- buf[:n]:
				break
			case <-remote.done:
				close(remote.channel)
				close(remote.done)
				return
			}
		}
	}()

	remote.channel = data

	return nil
}

// Channel return stdio channel
func (remote *StdioRemote) Channel() chan []byte {
	return remote.channel
}

// Close close stdio channel
func (remote *StdioRemote) Close() error {
	remote.done <- ""
	return nil
}

// Open open channel @deprecated
func (remote *StdioRemote) Open() chan []byte {
	return remote.channel
}

// Recv attempt to receive maximum amount of bytes within duration `t`
func (remote *StdioRemote) Recv(t time.Duration) ([]byte, error) {
	acc := make([]byte, 256)
	pos := 0

	for {
		select {
		case buf := <-remote.channel:
			if 0 == len(buf) {
				return nil, errors.New("EOF")
			}

			for i := 0; i < len(buf); i++ {
				acc[pos+i] = buf[i]
			}

			pos = pos + len(buf)

		case <-time.After(t):
			if 0 == pos {
				return []byte(""), errors.New("timeout")
			}

			log.Printf("stdio:recv[%v]: %v", pos, acc[:pos])
			return acc[:pos], nil
		}
	}
}

func (remote *StdioRemote) Write(buf []byte, timeout time.Duration) (int, error) {
	log.Printf("stdio:write[%v] %v\n", len(buf), buf)
	return remote.writer.Write(buf)
}
