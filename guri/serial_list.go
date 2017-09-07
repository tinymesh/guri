// +build !darwin

package guri

import (
	"fmt"
	"log"

	"go.bug.st/serial.v1/enumerator"
)

type SerialPort struct {
	Name         string
	IsUSB        bool
	VID          string
	PID          string
	SerialNumber string
}

// PortList return list of available serial ports
func PortList() ([]SerialPort, error) {
	ports, err := enumerator.GetDetailedPortsList()

	if err != nil {

		return nil, err
	}

	var results []SerialPort

	for _, portdef := range ports {
		var port = new(SerialPort)

		port.Name = portdef.Name
		port.IsUSB = portdef.IsUSB
		port.VID = portdef.VID
		port.PID = portdef.PID
		port.SerialNumber = portdef.SerialNumber

		results = append(results, *port)
	}

	return results, nil
}

// PrintPortList print ports to stdout
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
