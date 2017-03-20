package main

import (
	"fmt"
	"log"

	"go.bug.st/serial.v1/enumerator"
)

func PrintPortList() {
	ports, err := enumerator.GetDetailedPortsList()

	if err != nil {
		fmt.Println("enumerator.GetDetailedPortsList")
		log.Fatal(err)
	}

	if len(ports) == 0 {

	} else {
		for _, port := range ports {
			fmt.Printf("path=%v usb?=%v vid=%v pid=%v serial=%v\n",
				port.Name,
				port.IsUSB,
				port.VID,
				port.PID,
				port.SerialNumber,
			)
		}
	}
}
