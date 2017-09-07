// +build darwin

package guri

import (
	"fmt"
	"log"

	serial "go.bug.st/serial.v1"
)

// darwin needs IOKit to get GetDetailPortsList to work (which in turn required cgo, thus no
// cross-compiling atm)
func PrintPortList() {
	ports, err := serial.GetPortsList()

	if err != nil {
		fmt.Println("enumerator.GetDetailedPortsList")
		log.Fatal(err)
	}

	if len(ports) == 0 {

	} else {
		for _, port := range ports {
			fmt.Printf("path=%v usb?=%v vid=%v pid=%v serial=%v\n",
				port,
				nil,
				nil,
				nil,
				nil,
			)
		}
	}
}
