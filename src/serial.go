package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	serial "go.bug.st/serial.v1"
)

type SerialRemote struct {
	port    serial.Port
	channel chan []byte
}

func ConnectSerial(uri string) (*SerialRemote, error) {
	log.Printf("serial:open uri=%v\n", uri)
	port, err := serial.Open(uri, &serial.Mode{})

	if nil != err {
		return nil, err
	}

	mode := &serial.Mode{
		BaudRate: 19200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	if err := port.SetMode(mode); err != nil {
		return nil, err
	}

	return &SerialRemote{
		port:    port,
		channel: make(chan []byte, 256),
	}, nil
}

func (remote *SerialRemote) Open() chan []byte {
	go func() {
		acc := make([]byte, 256)
		var pos int = 0

		// remote.channel <- acc[:pos]
		reader := make(chan []byte)

		go func() {
			for {
				buf, err := remote.Recv(-1)

				if nil != err {
					log.Printf("error[serial:recv]: %v\n", err)
					close(reader)
					return
				} else if 0 == len(buf) {
					log.Printf("error[serial:recv]#EOF: %v\n", fmt.Errorf("EOF"))
					close(reader)
					return
				}

				reader <- buf
			}
		}()

		defer func() {
			if r := recover(); r != nil {
				log.Println("error[serial:close]:", r)
			}
		}()

		for {
			select {
			case buf, state := <-reader:
				if false == state {
					err := remote.Close()

					if nil != err {
						log.Printf("error[serial:close]: failed to close: %v\n", err)
					}

					close(remote.channel)
				} else {
					for i := 0; i < len(buf); i++ {
						acc[pos+i] = buf[i]
					}
					pos = pos + len(buf)
				}

				// @todo - make timeout match that of baudrate
			case <-time.After(3600 * time.Microsecond):
				if pos > 0 {
					var buf []byte = make([]byte, pos)
					copy(buf, acc[:pos])
					remote.channel <- buf
					pos = 0
				}
			}
		}
	}()

	return remote.channel
}

func (remote *SerialRemote) Channel() chan []byte {
	return remote.channel
}

func (remote *SerialRemote) Close() error {
	return remote.port.Close()
}

func (remote *SerialRemote) Recv(timeout time.Duration) ([]byte, error) {
	buf := make([]byte, 256)
	n, err := remote.port.Read(buf)

	if err != nil {
		return nil, err
	} else if 0 == n {
		return nil, errors.New("EOF")
	}
	//log.Printf("serial:recv %v\n", buf[:n])
	return buf[:n], nil
}

func (remote *SerialRemote) Write(buf []byte, timeout time.Duration) (int, error) {
	return remote.port.Write(buf)
}
