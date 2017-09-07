package guri

import (
	"crypto/tls"
	"errors"
	"log"
	"strings"
	"time"
)

// TLSConn information about TLS endpoint
type TLSConn struct {
	uri     string
	socket  tls.Conn
	channel chan []byte
}

// ConnectTLS connect to a TLS enabled enpoint
func ConnectTLS(uri string) (*TLSConn, error) {
	remote := &TLSConn{
		uri: uri,
	}

	if err := remote.Connect(); nil != err {
		return nil, err
	}

	return remote, nil
}

// Connect tls dialing
func (conn *TLSConn) Connect() error {
	log.Printf("tls:open uri=%v (SSL/TLS)\n", conn.uri)

	parts := strings.Split(conn.uri, ":")
	socket, err := tls.Dial("tcp", conn.uri, &tls.Config{
		ServerName: parts[0],
	})

	if err != nil {
		return err
	}

	conn.socket = *socket
	conn.channel = make(chan []byte)

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
				log.Printf("error[tcp/tls:recv] %v\n", err)
				conn.channel <- []byte("")
				close(conn.channel)
			} else {
				conn.channel <- buf[:n]
			}
		}
	}()

	return nil
}

// Channel return the TLS channel
func (conn *TLSConn) Channel() chan []byte {
	return conn.channel
}

// Close close TLS channel
func (conn *TLSConn) Close() error {
	return conn.socket.Close()
}

// Recv attempt to receive maximum amount of bytes within duration `t`
func (conn *TLSConn) Recv(t time.Duration) ([]byte, error) {
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

func (conn *TLSConn) Write(buf []byte, timeout time.Duration) (int, error) {
	log.Printf("tls:write[%v] %v\n", len(buf), buf)
	return conn.socket.Write(buf)
}
