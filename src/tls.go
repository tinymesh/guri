package main

import (
	"crypto/tls"
	"log"
	"strings"
	"time"
)

type TLSConn struct {
	socket  tls.Conn
	channel chan []byte
}

func ConnectTLS(uri string) (*TLSConn, error) {
	log.Printf("tls:open uri=%v (SSL/TLS)\n", uri)

	parts := strings.Split(uri, ":")
	socket, err := tls.Dial("tcp", uri, &tls.Config{
		ServerName: parts[0],
	})

	if err != nil {
		return nil, err
	}

	return &TLSConn{
		socket:  *socket,
		channel: make(chan []byte),
	}, nil
}

func (conn *TLSConn) Channel() chan []byte {
	return conn.channel
}

func (conn *TLSConn) Open() chan []byte {
	go func() {

		defer func() {
			if r := recover(); r != nil {
				log.Println("error[tcp/tls:close]:", r)
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

func (conn *TLSConn) Close() error {
	return conn.socket.Close()
}

func (conn *TLSConn) Recv(timeout time.Duration) ([]byte, error) {
	buf := make([]byte, 256)
	n, err := conn.socket.Read(buf)

	if nil != err {
		log.Printf("error[tcp/tls:recv] %v\n", err)
		return nil, err
	}

	return buf[:n], nil
}

func (conn *TLSConn) Write(buf []byte, timeout time.Duration) (int, error) {
	return conn.socket.Write(buf)
}
