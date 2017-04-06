package main

import (
	"errors"
	"log"
	"net"
	"time"
)

type TCPConn struct {
	uri     string
	socket  net.Conn
	channel chan []byte
}

func ConnectTCP(uri string) (*TCPConn, error) {
	remote := &TCPConn{
		uri: uri,
	}

	if err := remote.Connect(); nil != err {
		return nil, err
	}

	return remote, nil
}

func (conn *TCPConn) Connect() error {
	log.Printf("tcp:open uri=%v\n", conn.uri)

	socket, err := net.Dial("tcp", conn.uri)

	if err != nil {
		return err
	}

	conn.socket = socket
	conn.channel = make(chan []byte, 256)

	go func() {
		defer func() {
			if err := recover(); nil != err {
				log.Printf("error[serial] - %v", err)
			}
		}()

		buf := make([]byte, 256)

		for {
			n, err := conn.socket.Read(buf)

			if nil != err {
				log.Printf("error[tcp:recv] %v\n", err)
				conn.channel <- []byte("")
				close(conn.channel)
				return
			} else {
				conn.channel <- buf[:n]
			}
		}
	}()

	return nil
}

func (conn *TCPConn) Channel() chan []byte {
	return conn.channel
}

func (conn *TCPConn) Close() error {
	return conn.socket.Close()
}

func (conn *TCPConn) Recv(t time.Duration) ([]byte, error) {
	select {
	case buf := <-conn.channel:
		if 0 == len(buf) {
			return nil, errors.New("EOF")
		}

		return buf, nil

	case <-time.After(t):
		return []byte(""), nil
	}
}

func (conn *TCPConn) Write(buf []byte, timeout time.Duration) (int, error) {
	log.Printf("tcp:write[%v] %v\n", len(buf), buf)
	return conn.socket.Write(buf)
}
