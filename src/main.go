package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"
)

type Remote interface {
	Channel() chan []byte
	Recv(timeout time.Duration) ([]byte, error)
	Write(buf []byte, timeout time.Duration) (int, error)
	Open() chan []byte
	Close() error
}

func main() {
	helpFlag := flag.Bool("help", false, "Show help text")
	stdioFlag := flag.Bool("stdio", false, "Use stdio for communication instead of remote")
	listFlag := flag.Bool("list", false, "List available serialports")
	remoteFlag := flag.String("remote", "tcp.cloud.tiny-mesh.com:7002", "The upstream url to connect to")
	usetlsFlag := flag.Bool("tls", true, "Controll use of TLS with --remote")

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	flag.Parse()

	if true == *helpFlag {
		flag.PrintDefaults()
		return
	} else if true == *listFlag {
		PrintPortList()
		return
	}

	path := flag.Arg(0)

	if "" == path {
		log.Fatal(errors.New("1st argument, tty path, missing"))
	}

	var upstream Remote
	var downstream Remote
	var err error

	log.Printf("serial: opening %v\n", path)
	downstream, err = ConnectSerial(path)

	if nil != err {
		log.Fatal(err)
	}

	if true == *stdioFlag {
		log.Printf("remote: using stdio")
		upstream, err = ConnectStdio(os.Stdin, os.Stdout)

		if nil != err {
			log.Fatalf("error[stdio] %v\n", err)
		}
	} else if true == *usetlsFlag {
		// setup remote TLS communication
		log.Printf("remote: using TCP w/TLS")
		upstream, err = ConnectTLS(*remoteFlag)

		if nil != err {
			log.Fatalf("error[tcp/tls] %v\n", err)
		}
	} else {
		// setup remote TCP communication without TLS
		log.Printf("remote: using TCP (NO-TLS)")
		upstream, err = ConnectTCP(*remoteFlag)

		if nil != err {
			log.Fatalf("error[tcp] %v\n", err)
		}
	}

	if nil == upstream {
		log.Fatal(errors.New("no upstream configured"))
	}

	downstreamchan := downstream.Open()
	upstreamchan := upstream.Open()

	for {
		select {
		case buf := <-downstreamchan:
			log.Printf("downstream:recv %v\n", buf)
			upstream.Write(buf, -1)

		case buf := <-upstreamchan:
			log.Printf("upstream:recv %v\n", buf)
			if true == *stdioFlag && buf[0] == 10 {
				downstream.Write([]byte("\x0a\x00\x00\x00\x00\x03\x03\x10\x00\x00"), -1)
			}

			downstream.Write(buf, -1)
		}
	}
}
