package main

import (
	"errors"
	"log"
)

func GetNIDCmd(addr Address) []byte {
	return []byte{10, 0, 0, 0, 0, 0, 3, 16, 0, 0}
}
func SetGwConfigModeCmd(addr Address) []byte {
	return []byte{10, addr[0], addr[1], addr[2], addr[3], 0, 3, 5, 0, 0}
}

type GenericEvent struct {
	uid         Address
	sid         Address
	rssi        byte
	network_lvl byte
	hops        byte
	packet_num  uint16
	latency     uint16
	packettype  byte
	detail      byte
	data        []byte
	address     Address
	temp        byte
	volt        float32
	digitalIO   byte
	aio0        []byte
	aio1        []byte
	hwrevision  []byte
	fwrevision  []byte
}

func decode(buf []byte) (GenericEvent, error) {
	if len(buf) != 35 {
		return GenericEvent{}, errors.New("expected a generic Tinymesh event, incomplete")
	} else if buf[0] != 35 {
		return GenericEvent{}, errors.New("expected a generic Tinymesh event, length invalid")
	} else if buf[16] != 2 {
		return GenericEvent{}, errors.New("expected a generic Tinymesh event, packetType /= 2")
	}

	return GenericEvent{
		sid:         buf[1:5],
		uid:         buf[5:9],
		rssi:        buf[9],
		network_lvl: buf[10],
		hops:        buf[11],
		packet_num:  uint16(buf[12]<<8) + uint16(buf[13]),
		latency:     uint16(buf[14]<<8) + uint16(buf[15]),
		packettype:  buf[16],
		detail:      buf[17],
		data:        buf[18:20],
		address:     buf[20:24],
		temp:        buf[24] - 128,
		volt:        float32(buf[25]) * 0.030,
		digitalIO:   buf[26],
		aio0:        buf[27:29],
		aio1:        buf[29:31],
		hwrevision:  buf[31:33],
		fwrevision:  buf[33:35],
	}, nil
}

type ConfigValue []byte

func RunConfigCmd(remote Remote, cmd byte, waitForPrompt bool) error {
	var err error

	if _, err = remote.Write([]byte{cmd}, -1); err != nil {
		return err
	}

	if waitForPrompt {
		_ = WaitForConfig(remote)
	}

	return nil
}

func SetConfigurationMemory(remote Remote, pairs []ConfigValue) error {
	var err error

	if _, err = remote.Write([]byte{'M'}, -1); err != nil {
		return err
	}

	_ = WaitForConfig(remote)

	log.Printf("tinymesh:config: %v\n", pairs)

	for _, pair := range pairs {
		_, err := remote.Write([]byte{pair[0], pair[1]}, -1)

		if err != nil {
			return err
		}
	}

	if _, err = remote.Write([]byte{255}, -1); err != nil {
		return err
	}

	_ = WaitForConfig(remote)

	return nil
}

func SetCalibrationMemory(remote Remote, pairs []ConfigValue) error {
	var err error

	if _, err = remote.Write([]byte{'H', 'W'}, -1); err != nil {
		return err
	}

	_ = WaitForConfig(remote)

	log.Printf("tinymesh:calibration: %v\n", pairs)

	for _, pair := range pairs {
		_, err := remote.Write([]byte{pair[0], pair[1]}, -1)

		if err != nil {
			return err
		}
	}

	if _, err = remote.Write([]byte{255}, -1); err != nil {
		return err
	}

	_ = WaitForConfig(remote)

	return nil
}
