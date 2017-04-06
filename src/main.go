package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

var (
	vsn = "0.0.1-rc2"
)

type Remote interface {
	Channel() chan []byte
	Close() error
	Connect() error
	Recv(timeout time.Duration) ([]byte, error)
	Write(buf []byte, timeout time.Duration) (int, error)
}

type Flags struct {
	help    bool
	list    bool
	version bool

	verify        bool
	nid           Address
	sid           Address
	uid           Address
	autoConfigure bool

	stdio     bool
	remote    string
	tls       bool
	reconnect bool
}

func parseFlags() Flags {
	flags := new(Flags)

	// commands
	listFlag := flag.Bool("list", false, "List available serialports")
	helpFlag := flag.Bool("help", false, "Show help text")
	versionFlag := flag.Bool("version", false, "Show version")

	// link flags
	verifyFlag := flag.Bool("verify", true, "validate IDs according to -nid, -sid, and -uid flags")
	autoConfigureFlag := flag.Bool("auto-configure", false, "Automatically configure gateway operation and ID's; use -nid, -sid, and -uid flags")
	nidFlag := flag.String("nid", "::", "32bit Network ID in hexadecimal (ie, aa:bb:cc:dd)")
	sidFlag := flag.String("sid", "::", "32bit System ID in hexadecimal (ie, aa:bb:cc:dd)")
	uidFlag := flag.String("uid", "::", "32bit Unique ID in hexadecimal (ie, aa:bb:cc:dd)")

	// communication flags
	stdioFlag := flag.Bool("stdio", false, "Use stdio for communication instead of remote")
	remoteFlag := flag.String("remote", "tcp.cloud.tiny-mesh.com:7002", "The upstream url to connect to")
	usetlsFlag := flag.Bool("tls", true, "Controll use of TLS with -remote")
	reconnectFlag := flag.Bool("reconnect", true, "Automatically re-establish communication on failure")

	flag.Parse()

	flags.help = *helpFlag
	flags.list = *listFlag
	flags.version = *versionFlag

	flags.verify = *verifyFlag
	flags.autoConfigure = *autoConfigureFlag
	flags.nid = parseAddr(*nidFlag)
	flags.sid = parseAddr(*sidFlag)
	flags.uid = parseAddr(*uidFlag)

	if len(flags.nid) == 0 {
		log.Fatalf("failed to parse -nid value, value must be 4 bytes encoded as hexadecimals with : as a separator\nexample: -nid 01:02:03:04\n")
	}
	if len(flags.sid) == 0 {
		log.Fatalf("failed to parse -sid value, value must be 4 bytes encoded as hexadecimals with : as a separator\nexample: -sid 01:02:03:04\n")
	}
	if len(flags.uid) == 0 {
		log.Fatalf("failed to parse -uid value, value must be 4 bytes encoded as hexadecimals with : as a separator\nexample: -uid 01:02:03:04\n")
	}

	flags.stdio = *stdioFlag
	flags.remote = *remoteFlag
	flags.tls = *usetlsFlag
	flags.reconnect = *reconnectFlag

	return *flags
}

func pickUpstream(flags Flags) (Remote, error) {
	if true == flags.stdio {
		// stdio
		return ConnectStdio(os.Stdin, os.Stdout)
	} else if true == flags.tls {
		// tls
		return ConnectTLS(flags.remote)
	}

	return ConnectTCP(flags.remote)
}

func forward(remote Remote, t time.Duration) chan []byte {
	ch := make(chan []byte, 256)

	go func() {
		for {
			buf, err := remote.Recv(t)

			if nil != err {
				log.Printf("forward-err: %v\n", err)
				close(ch)
				return
			}

			ch <- buf
		}
	}()

	return ch
}

func loop(from Remote, to Remote, flags Flags) {
	upstream := forward(from, 500*time.Millisecond)
	downstream := forward(to, 2*time.Millisecond)

	upoff := &Backoff{
		initial: 1 * time.Second,
		wait:    1 * time.Second,
		delay:   2.5,
		max:     5 * time.Minute,
	}

	downoff := &Backoff{
		initial: 1 * time.Second,
		wait:    1 * time.Second,
		delay:   2.5,
		max:     5 * time.Minute,
	}

	for {
		select {
		case buf, state := <-upstream:
			if false == state {
				if !flags.reconnect {
					log.Fatalf("upstream:close, exiting")
				}

				log.Printf("upstream:close, reconnecting\n")
				from.Close()
				if err := from.Connect(); nil != err {
					log.Printf("upstream:open: %v\n", err)
					upoff.Fail()
				} else {
					upoff.Success()
					upstream = forward(from, 500*time.Millisecond)
				}
			} else if len(buf) > 0 {
				log.Printf("upstream:recv[%v] %v\n", state, buf)
				if len(buf) > 10 && 6 == buf[0] {
					to.Write(buf[:1], -1)
					to.Write(buf[1:], -1)
				} else {
					to.Write(buf, -1)
				}
			}

		case buf, state := <-downstream:
			if false == state {
				if !flags.reconnect {
					log.Fatalf("upstream:close, exiting")
				}

				to.Close()
				log.Printf("downstream:close, reconnecting\n")
				if err := to.Connect(); nil != err {
					log.Printf("downstream:open: %v\n", err)
					downoff.Fail()
				} else {
					downoff.Success()
					downstream = forward(to, 2*time.Millisecond)
				}
			} else if len(buf) > 0 {
				log.Printf("downstream:recv[%v] %v\n", state, buf)
				from.Write(buf, -1)
			}
		}
	}
}

func main() {

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	flags := parseFlags()

	if true == flags.help {
		flag.PrintDefaults()
		return
	} else if true == flags.list {
		PrintPortList()
		return
	} else if true == flags.version {
		fmt.Printf("%v\n", vsn)
		return
	}

	path := flag.Arg(0)

	if "" == path {
		log.Fatal(errors.New("1st argument, tty path, missing"))
	}

	var upstream Remote
	var downstream Remote
	var err error

	log.Printf("guri - version %v\n", vsn)

	if downstream, err = ConnectSerial(path, flags); nil != err {
		log.Fatal(err)
	}

	if upstream, err = pickUpstream(flags); nil != err {
		log.Fatalf("failed to connect to upstream; %v\n", err)
	} else {
		loop(upstream, downstream, flags)
	}
}
