package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
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
			fmt.Printf("port: %v\n", port)
		}
	}
}

func Open(path string) serial.Port {
	port, err := serial.Open(path, &serial.Mode{})

	if err != nil {
		fmt.Fprintln(os.Stderr, "port.Open")
		log.Fatal(err)
	}

	mode := &serial.Mode{
		BaudRate: 19200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	if err := port.SetMode(mode); err != nil {
		fmt.Fprintln(os.Stderr, "port.SetMode")
		log.Fatal(err)
	}

	return port
}

func RecvIO(port io.Reader) chan []byte {
	channel := make(chan []byte, 256)
	input := make([]byte, 256+2)
	go func() {
		for {
			n, err := io.ReadAtLeast(port, input, 1)

			if n == 0 {
				fmt.Fprintln(os.Stderr, "STDIN: EOF")
			}

			if err != nil {
				fmt.Fprintln(os.Stderr, "io.ReadAtLeast")
				log.Fatal(err)
				break
			}

			channel <- input[0:n]
		}
	}()

	return channel
}

func RecvSerial(port serial.Port) chan []byte {
	channel := make(chan []byte, 256)

	go func() {
		acc := make([]byte, 256)
		input := make([]byte, 256)
		var pos int = 0

		// spawn a reader serialport reader
		reader := make(chan int)
		go func() {
			for {
				n, err := port.Read(input)

				// error or EOF
				if err != nil || n == 0 {
					log.Printf("Error in serial port reader:\n")
					if err != nil {
						log.Print(err)
					} else {
						log.Print("EOF")
					}

					reader <- -1
					port.Close()
					close(reader)

					return
				}

				reader <- n
			}
		}()

		for {
			select {
			case n := <-reader:
				// got data or canceled channel
				if n == -1 {
					close(channel)
					return
				}

				for i := 0; i < n; i++ {
					acc[pos+i] = input[i]
				}
				pos = pos + n

			case <-time.After(2400 * time.Microsecond):
				if pos > 0 {
					channel <- acc[:pos]
					pos = 0
				}
			}
		}
	}()

	return channel
}

// - [--usage]: print help
// - <--list>: list serial ports
// - <port>: connect to the port
func main() {
	args := os.Args

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	if len(args) == 1 || "--usage" == args[1] {
		fmt.Printf("usage: %s --usage | --list | <port>\n", path.Dir(args[0]))
	} else if "--list" == args[1] {
		PrintPortList()
	} else {
		// connect to port
		port := Open(args[1])
		defer port.Close()

		status, err := port.GetModemStatusBits()
		if err != nil {
			fmt.Fprintln(os.Stderr, "port.GetModemStatusBits")
			log.Fatal(err)
		}
		fmt.Fprintln(os.Stderr, "port-status: %+v\n", status)

		upstream := RecvIO(os.Stdin)
		serial := RecvSerial(port)

		for {
			select {
			case buf := <-serial:
				if nil != buf {
					WriteIO(os.Stdout, buf)
				} else {
					log.Fatal("serialport communication failed, exiting...")
				}

			case buf := <-upstream:
				if buf[0] == 10 {
					WriteSerial(port, []byte("\x0a\x00\x00\x00\x00\x03\x03\x10\x00\x00"))
				} else {
					WriteSerial(port, buf)
				}
			}
		}
	}

}

func WriteIO(writer io.Writer, buf []byte) (int, error) {
	log.Printf("write:stdin: %v\n", buf)
	return writer.Write(buf)
}

func WriteSerial(port serial.Port, buf []byte) (int, error) {
	log.Printf("write:serial: %v\n", buf)
	return port.Write(buf)
}
