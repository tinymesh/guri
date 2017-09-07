package guri

import (
	"time"
)

// Remote ...
type Remote interface {
	Channel() chan []byte
	Close() error
	Connect() error
	Recv(timeout time.Duration) ([]byte, error)
	Write(buf []byte, timeout time.Duration) (int, error)
}

// Flags ...
type Flags struct {
	Help    bool
	List    bool
	Version bool

	Verify        bool
	NID           Address
	SID           Address
	UID           Address
	AutoConfigure bool

	Stdio     bool
	Remote    string
	TLS       bool
	Reconnect bool
}
