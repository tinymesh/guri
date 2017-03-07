package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go.bug.st/serial.v1"
)

func PrintPortList() {
	ports, err := serial.GetPortsList()

	if err != nil {
		fmt.Println("serial.GetPortsList")
		log.Fatal(err)
	}

	if len(ports) == 0 {
		fmt.Println("No serial ports found!")
	} else {
		for _, port := range ports {
			fmt.Printf("%v\n", port)
		}
	}
}

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

	} else {
		// setup remote communication
		log.Fatal(errors.New("--remote not implemented"))
	}

	if nil == upstream {
		log.Fatal(errors.New("no upstream configured"))
	}

	downstreamchan := downstream.Open()
	upstreamchan := upstream.Open()

	for {
		select {
		case buf := <-downstreamchan:
			upstream.Write(buf, -1)

		case buf := <-upstreamchan:
			if buf[0] == 10 {
				downstream.Write([]byte("\x0a\x00\x00\x00\x00\x03\x03\x10\x00\x00"), -1)
			}

			downstream.Write(buf, -1)
		}
	}
}
