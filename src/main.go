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
	vsn = "0.0.1-alpha"
)

type Remote interface {
	Channel() chan []byte
	Recv(timeout time.Duration) ([]byte, error)
	Write(buf []byte, timeout time.Duration) (int, error)
	Open() chan []byte
	Close() error
}

type Flags struct {
	help    bool
	list    bool
	version bool

	verify bool
	nid    Address
	sid    Address
	uid    Address

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
	verifyFlag := flag.Bool("verify", false, "validate IDs according to --nid, --sid, and --uid flags")

	nidFlag := flag.String("nid", "::", "32bit Network ID in hexadecimal (ie, aa:bb:cc:dd)")
	sidFlag := flag.String("sid", "::", "32bit System ID in hexadecimal (ie, aa:bb:cc:dd)")
	uidFlag := flag.String("uid", "::", "32bit Unique ID in hexadecimal (ie, aa:bb:cc:dd)")

	// communication flags
	stdioFlag := flag.Bool("stdio", false, "Use stdio for communication instead of remote")
	remoteFlag := flag.String("remote", "tcp.cloud.tiny-mesh.com:7002", "The upstream url to connect to")
	usetlsFlag := flag.Bool("tls", true, "Controll use of TLS with --remote")
	reconnectFlag := flag.Bool("reconnect", false, "Automatically reconnect upstream (tcp,tls) on failure")

	flag.Parse()

	flags.help = *helpFlag
	flags.list = *listFlag
	flags.version = *versionFlag

	flags.verify = *verifyFlag
	flags.nid = parseAddr(*nidFlag)
	flags.sid = parseAddr(*sidFlag)
	flags.uid = parseAddr(*uidFlag)

	flags.stdio = *stdioFlag
	flags.remote = *remoteFlag
	flags.tls = *usetlsFlag
	flags.reconnect = *reconnectFlag

	return *flags
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
	log.Printf("serial: opening %v\n", path)
	downstream, err = ConnectSerial(path)

	if nil != err {
		log.Fatal(err)
	}

	downstreamchan := downstream.Open()

	// check --verify
	if flags.verify {
		_, err = downstream.Write([]byte{255}, -1)

		if nil != err {
			log.Fatal("failed to check config mode state: ", err)
		}

		select {
		case configPrompt := <-downstreamchan:
			if configPrompt[0] == '>' {
				// in config mode
				log.Fatal("Device in config mode... you should manually exit\n")
			}
			break

		case <-time.After(500 * time.Millisecond):
			break
		}

		_, err = downstream.Write(GetNID([]byte{0, 0, 0, 0}), -1)

		select {
		case nidEv := <-downstreamchan:
			ev, err := decode(nidEv)

			if err != nil {
				log.Fatal(err)
			}

			if !flags.nid.Equal(ev.address) {
				log.Fatalf("failed to verify Network ID (%v vs %v)", flags.nid.ToString(), ev.address.ToString())
			} else if !flags.nid.Equal(ev.address) {
				log.Fatalf("failed to verify System ID (%v vs %v)", flags.sid.ToString(), ev.sid.ToString())
			} else if !flags.uid.Equal(ev.uid) {
				log.Fatalf("failed to verify Unique ID (%v vs %v)", flags.uid.ToString(), ev.uid.ToString())
			}

			break

		case <-time.After(500 * time.Millisecond):
			log.Fatal("failed to request NID: ", "timeout")
			break
		}
	}

	connectUpstream := func() (Remote, error) {

		if true == flags.stdio {
			log.Printf("remote: using stdio")
			upstream, err = ConnectStdio(os.Stdin, os.Stdout)

			if nil != err {
				log.Fatalf("error[stdio] %v\n", err)
			}
		} else if true == flags.tls {
			// setup remote TLS communication
			log.Printf("remote: using TCP w/TLS")
			upstream, err = ConnectTLS(flags.remote)

			if nil != err {
				log.Printf("error[tcp/tls] %v\n", err)
				return nil, err
			}
		} else {
			// setup remote TCP communication without TLS
			log.Printf("remote: using TCP (NO-TLS)")
			upstream, err = ConnectTCP(flags.remote)

			if nil != err {
				log.Printf("error[tcp] %v\n", err)
				return nil, err
			}
		}

		if nil == upstream {
			return nil, errors.New("no upstream configured")
		}

		return upstream, nil
	}

	upstream, err = connectUpstream()

	// downstreamchan := downstream.Open()
	upstreamchan := upstream.Open()

	var maxRetries = 0

	if true == flags.reconnect {
		maxRetries = -1
	}

	var upstreamBackoff Backoff = NewBackoff(time.Second, 2.0, maxRetries)
	var downstreamBackoff Backoff = NewBackoff(time.Second, 2.0, maxRetries)

	for {
		select {
		case buf, state := <-downstreamchan:
			if false == state {
				if true == flags.reconnect {
					log.Printf("downstream:close, reconnecting\n")
					downstreamBackoff.Until(func() error {
						downstream, err = ConnectSerial(path)
						if nil != err {
							log.Printf("error[downstream:connect] %v\n", err)
						} else {
							log.Printf("downstream:connect reconnected\n")
						}
						return err
					})

					downstreamchan = downstream.Open()
				} else {
					log.Printf("downstream:close, terminating\n")
					return
				}
			} else {
				log.Printf("downstream:recv[%v] %v\n", state, buf)
				upstream.Write(buf, -1)
			}

		case buf, state := <-upstreamchan:
			if false == state {
				if true == flags.reconnect {
					log.Printf("upstream:close, reconnecting\n")
					upstreamBackoff.Until(func() error {
						upstream, err = connectUpstream()
						if nil != err {
							log.Printf("error[upstream:connect] %v\n", err)
						} else {
							log.Printf("upstream:connect reconnect\n")
						}
						return err
					})

					upstreamchan = upstream.Open()
				} else {
					log.Printf("upstream:close, terminating\n")
					return
				}

			} else {
				log.Printf("upstream:recv[%v] %v\n", state, buf)
				if true == flags.stdio && buf[0] == 10 {
					downstream.Write([]byte("\x0a\x00\x00\x00\x00\x03\x03\x10\x00\x00"), -1)
				}

				downstream.Write(buf, -1)
			}
		}
	}
}
