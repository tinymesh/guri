package main

import (
	"fmt"
	"log"
	"time"
)

// timeout
const byteTimeout = 1000000 / 19200 / 10 * time.Microsecond

// check if we are in conif mode
func inTinyMeshConfig(remote *SerialRemote) (bool, error) {
	bytes, err := remote.Write([]byte{255, 255, 255}, -1)

	if nil != err {
		return true, err
	} else if 3 != bytes {
		return true, fmt.Errorf("serial:config/inTinyMeshConfig: config mode check failed, unable to write 3 bytes")
	}

	// open a channel that evetually will return some data
	// buf, err := readUntilTimeout(port, byteTimeout*2*time.Microsecond)
	buf, err := remote.Recv(byteTimeout * 2 * time.Microsecond)

	if nil == buf {
		// if we can't get a response, assume we are in config and do nothing
		return true, err
	} else {
		return len(buf) > 0 && '>' == buf[0], nil
	}
}

func WaitForTinyMeshConfig(remote *SerialRemote) error {
	inCfg, err := inTinyMeshConfig(remote)

	if nil != err {
		return err
	} else if true == inCfg {
		return nil
	}

	_, err = remote.Write(SetGwConfigModeCmd([]byte{0, 0, 0, 0}), -1)

	if nil != err {
		return err
	}

	inCfg, err = inTinyMeshConfig(remote)

	if nil != err {
		return err
	} else if true == inCfg {
		return nil
	}

	log.Printf("!! Press configuration button to continue\n")

	for {
		buf, err := remote.Recv(50 * time.Millisecond)

		if nil == buf {
			return err
		} else {
			if len(buf) > 0 && '>' == buf[0] {
				return nil
			}
		}
	}
}

func verifyTinyMeshConfig(remote *SerialRemote, flags Flags) error {
	inCfg, err := inTinyMeshConfig(remote)

	if nil != err {
		return err
	}
	//
	if true == inCfg {
		_, err := remote.Write([]byte("X"), -1)

		inCfg, err = inTinyMeshConfig(remote)

		if nil != err {
			return err
		} else if true == inCfg {
			return fmt.Errorf("serial:config/verifyTinyMeshConfig: failed to exit config mode\n")
		}
	}

	// ask for a NID, 1.92 bytes pr. ms
	tries := 0
	for {
		if _, err = remote.Write(GetNIDCmd([]byte{0, 0, 0, 0}), -1); nil != err {
			return err
		}

		// open a channel that evetually will at wait 2 millisecond since last read
		// to return
		buf, err := remote.Recv(2 * time.Millisecond)

		if nil != buf {
			ev, err := decode(buf)
			if nil == err && 18 == ev.detail {
				if !flags.nid.Equal(ev.address) {
					return fmt.Errorf("serial:config: failed to verify Network ID (%v vs %v)", flags.nid.ToString(), ev.address.ToString())
				} else if !flags.sid.Equal(ev.sid) {
					return fmt.Errorf("serial:config: failed to verify System ID (%v vs %v)", flags.sid.ToString(), ev.sid.ToString())
				} else if !flags.uid.Equal(ev.uid) {
					return fmt.Errorf("serial:config: failed to verify Unique ID (%v vs %v)", flags.uid.ToString(), ev.uid.ToString())
				}

				return nil

			} else if nil != err || 18 != ev.detail {

				tries = tries + 1

				if tries >= 3 {
					break
				}
			}
		} else if nil != err {
			return err
		}
	}

	return fmt.Errorf("should never get herssse")
}

func ensureTinyMeshConfig(remote *SerialRemote, flags Flags) error {
	// If verifyication is successfull it means we are a gateway with whatever
	// options specified in flags
	var err error

	if err = WaitForTinyMeshConfig(remote); nil != err {
		return err
	}

	if err = RunConfigCmd(remote, '0', false); err != nil {
		log.Fatalf("serial:config: failed to request configuration memory: %v", err)
	}

	cfg, err := remote.Recv(255 * time.Millisecond)
	if nil != err {
		log.Fatalf("serial:config: failed to read configuration memory: %v", err)
	}

	if err = RunConfigCmd(remote, 'r', false); err != nil {
		log.Fatalf("serial:config: failed to request calibration memory: %v", err)
	}

	cal, err := remote.Recv(255 * time.Millisecond)
	if nil != err {
		fmt.Errorf("serial:config: failed to read calibration memory: %v", err)
	}

	usingProtocol := cfg[3]
	deviceType := cfg[14]
	uid := cfg[45:49]
	sid := cfg[49:53]
	nid := cal[23:27]

	log.Printf("serial:config: protocol=%v deviceType=%v uid=%v sid=%v nid=%v",
		usingProtocol,
		deviceType,
		AddressToString(uid),
		AddressToString(sid),
		AddressToString(nid))

	if 1 != deviceType {
		log.Println("serial:config: ensure gateway operations")
		if err = RunConfigCmd(remote, 'G', true); err != nil {
			log.Fatalf("serial:config: failed to enable gateway mode: %v", err)
		}
	}

	newCfg := []ConfigValue{}

	if 0 != usingProtocol {
		newCfg = append(newCfg, ConfigValue{3, 0})
	}

	if !flags.uid.Equal(uid) {
		newCfg = append(newCfg, ConfigValue{45, flags.uid[0]})
		newCfg = append(newCfg, ConfigValue{46, flags.uid[1]})
		newCfg = append(newCfg, ConfigValue{47, flags.uid[2]})
		newCfg = append(newCfg, ConfigValue{48, flags.uid[3]})
	}

	if !flags.sid.Equal(sid) {
		newCfg = append(newCfg, ConfigValue{49, flags.sid[0]})
		newCfg = append(newCfg, ConfigValue{50, flags.sid[1]})
		newCfg = append(newCfg, ConfigValue{51, flags.sid[2]})
		newCfg = append(newCfg, ConfigValue{52, flags.sid[3]})
	}

	if len(newCfg) > 0 {
		log.Println("serial:config: set configuration")
		if err = SetConfigurationMemory(remote, newCfg); err != nil {
			log.Fatalf("serial:config:failed to set configuration memory: %v\n :: %v\n", newCfg, err)
		}
	}

	if !flags.nid.Equal(nid) {
		setNid := []ConfigValue{
			ConfigValue{23, flags.nid[0]},
			ConfigValue{24, flags.nid[1]},
			ConfigValue{25, flags.nid[2]},
			ConfigValue{26, flags.nid[3]},
		}

		log.Println("serial:config: set calibration")
		if err = SetCalibrationMemory(remote, setNid); err != nil {
			log.Fatalf("serial:config:failed to set calibration memory: %v\n :: %v\n", setNid, err)
		}
	}

	if err = RunConfigCmd(remote, 'X', false); err != nil {
		log.Fatalf("serial:config: failed to exit configuration mode: %v", err)
	}

	return nil
}
