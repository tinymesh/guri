package main

import (
	"errors"
	"log"
	"time"

	serial "go.bug.st/serial.v1"
)

type SerialRemote struct {
	uri   string
	flags Flags
	port  serial.Port
	// data channel
	channel chan []byte
	// done channel, send something to close it
	done chan struct{}
}

func ConnectSerial(uri string, flags Flags) (*SerialRemote, error) {
	remote := &SerialRemote{
		uri:   uri,
		flags: flags,
	}

	err := remote.Connect()

	if nil != err {
		return nil, err
	}

	return remote, nil
}

func (remote *SerialRemote) Connect() error {
	log.Printf("serial:open uri=%v\n", remote.uri)

	port, err := serial.Open(remote.uri, &serial.Mode{})

	if nil != err {
		return err
	}

	mode := &serial.Mode{
		BaudRate: 19200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	if err := port.SetMode(mode); err != nil {
		return err
	}

	remote.port = port
	remote.done = make(chan struct{}, 2)
	remote.channel = make(chan []byte, 256)

	go remote.ioloop()

	if true == remote.flags.autoConfigure {
		// configureGateway this, flags
		if err := ensureTinyMeshConfig(remote, remote.flags); nil != err {
			return err
		}
	} else if true == remote.flags.verify {
		if err := verifyTinyMeshConfig(remote, remote.flags); nil != err {
			return err
		}
	}

	return nil
}

func (remote *SerialRemote) SetState(state bool) error {
	return remote.port.SetRTS(state)
}

func (remote *SerialRemote) ioloop() {
	defer func() {
		defer func() {
			if err := recover(); nil != err {
				log.Printf("error[serial] - %v", err)
			}
		}()

		close(remote.done)
	}()

	for {
		buf := make([]byte, 256)
		bytes, err := remote.port.Read(buf)

		if nil != err {
			log.Printf("serial:err - failed to read port %v\n", err)
			remote.channel <- []byte("")
			return
		} else if 0 == bytes {
			remote.channel <- []byte("")
			return
		}

		select {
		case remote.channel <- buf[:bytes]:
			break
		case <-remote.done:
			remote.channel <- []byte("")
			return
		}
	}
}

func (remote *SerialRemote) Channel() chan []byte {
	return remote.channel
}

func (remote *SerialRemote) Close() error {
	return remote.port.Close()
}

func (remote *SerialRemote) Recv(t time.Duration) ([]byte, error) {
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
				return []byte(""), nil
			}

			return acc[:pos], nil
		}
	}
}

func (remote *SerialRemote) Write(buf []byte, timeout time.Duration) (int, error) {
	log.Printf("serial:write[%v]: %v\n", len(buf), buf)
	bytes, err := remote.port.Write(buf)
	time.Sleep(UARTTimeout() * 2)

	return bytes, err
}
