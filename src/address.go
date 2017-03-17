package main

import (
	"bytes"
	"fmt"
)

type Address []byte

const AddressLength = 4

func (address Address) Equal(match Address) bool {
	if bytes.Equal(address, []byte{0, 0, 0, 0}) {
		return true
	}
	return bytes.Equal(address, match)
}

func (addr Address) ToString() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x", addr[0], addr[1], addr[2], addr[3])
}

// adapted from golang/net.parse
func parseAddr(s string) Address {
	addr := make(Address, AddressLength)
	ellipsis := -1 // position of ellipsis in ip

	// Might have leading ellipsis
	if len(s) >= 2 && s[0] == ':' && s[1] == ':' {
		ellipsis = 0
		s = s[2:]
		// Might be only ellipsis
		if len(s) == 0 {
			return addr
		}
	}

	// Loop, parsing hex numbers followed by colon.
	i := 0
	for i < AddressLength {
		// Hex number.
		n, c, ok := xtoi(s)
		if !ok || n > 0xFF {
			return nil
		}

		// Save this byte
		addr[i] = byte(n)
		i += 1

		// Stop at end of string.
		s = s[c:]
		if len(s) == 0 {
			break
		}

		// Otherwise must be followed by colon and more.
		if s[0] != ':' || len(s) == 1 {
			return nil
		}
		s = s[1:]

		// Look for ellipsis.
		if s[0] == ':' {
			if ellipsis >= 0 { // already have one
				return nil
			}
			ellipsis = i
			s = s[1:]
			if len(s) == 0 { // can be at end
				break
			}
		}
	}

	// Must have used entire string.
	if len(s) != 0 {
		return nil
	}

	n := AddressLength - i
	for j := i - 1; j >= ellipsis; j-- {
		addr[j+n] = addr[j]
	}
	for j := ellipsis + n - 1; j >= ellipsis; j-- {
		addr[j] = 0
	}

	return addr
}

// Returns number, characters consumed, success. from Golang source code
const big = 0xFFFFFF

func xtoi(s string) (n int, i int, ok bool) {
	n = 0
	for i = 0; i < len(s); i++ {
		if '0' <= s[i] && s[i] <= '9' {
			n *= 16
			n += int(s[i] - '0')
		} else if 'a' <= s[i] && s[i] <= 'f' {
			n *= 16
			n += int(s[i]-'a') + 10
		} else if 'A' <= s[i] && s[i] <= 'F' {
			n *= 16
			n += int(s[i]-'A') + 10
		} else {
			break
		}
		if n >= big {
			return 0, i, false
		}
	}
	if i == 0 {
		return 0, i, false
	}
	return n, i, true
}
