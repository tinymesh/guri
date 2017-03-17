package main

import "errors"

func GetNID(addr Address) []byte {
	return []byte{10, 0, 0, 0, 0, 0, 3, 16, 0, 0}
}
func SetGwConfigMode(addr Address) []byte {
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
		uid:         buf[1:5],
		sid:         buf[5:9],
		rssi:        buf[9],
		network_lvl: buf[10],
		hops:        buf[11],
		packet_num:  uint16(buf[13] + buf[12]<<8),
		latency:     uint16(buf[15] + buf[14]<<8),
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
