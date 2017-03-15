package main

import (
	"log"
	"net"
	"time"
)

type TCPConn struct {
	socket  net.Conn
	channel chan []byte
}

func ConnectTCP(uri string) (*TCPConn, error) {
	socket, err := net.Dial("tcp", uri)

	if err != nil {
		return nil, err
	}

	return &TCPConn{
		socket:  socket,
		channel: make(chan []byte, 256),
	}, nil
}

func (conn *TCPConn) Channel() chan []byte {
	return conn.channel
}

func (conn *TCPConn) Open() chan []byte {
	go func() {

		defer func() {
			if r := recover(); r != nil {
				log.Println("error[tcp:close]:", r)
			}
		}()

		for {
			buf, err := conn.Recv(-1)

			if nil != err {
				close(conn.channel)
			} else {
				conn.channel <- buf
			}
		}
	}()

	return conn.Channel()
}

func (conn *TCPConn) Close() error {
	return conn.socket.Close()
}

func (conn *TCPConn) Recv(timeout time.Duration) ([]byte, error) {
	buf := make([]byte, 256)
	n, err := conn.socket.Read(buf)

	if nil != err {
		log.Printf("error[tcp:recv] %v\n", err)
		return nil, err
	}

	return buf[:n], nil
}

func (conn *TCPConn) Write(buf []byte, timeout time.Duration) (int, error) {
	return conn.socket.Write(buf)
}
