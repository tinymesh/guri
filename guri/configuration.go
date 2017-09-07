package guri

import (
	"fmt"
	"log"
	"time"
)

func remoteInConfigMode(remote Remote) (bool, error) {
	_, err := remote.Write([]byte{255}, -1)

	if nil != err {
		return false, fmt.Errorf("failed to check config mode state: %v", err)
	}

	select {
	case configPrompt := <-remote.Channel():
		if configPrompt[0] == '>' {
			// in config mode
			return true, nil
		}
		break

	case <-time.After(500 * time.Millisecond):
		break
	}

	return false, nil
}

func verifyIDs(remote Remote, flags Flags) error {
	configMode, err := remoteInConfigMode(remote)
	if nil != err {
		return err
	} else if configMode {
		return fmt.Errorf("main:config: Device in config mode... you must exit manually")
	}

	_, _ = remote.Write(GetNIDCmd([]byte{0, 0, 0, 0}), -1)

	select {
	case nidEv := <-remote.Channel():
		ev, err := decode(nidEv)

		if err != nil {
			log.Fatal(err)
		}

		if !flags.NID.Equal(ev.address) {
			return fmt.Errorf("main:config: failed to verify Network ID (%v vs %v)", flags.NID.ToString(), ev.address.ToString())
		} else if !flags.SID.Equal(ev.sid) {
			return fmt.Errorf("main:config: failed to verify System ID (%v vs %v)", flags.SID.ToString(), ev.sid.ToString())
		} else if !flags.UID.Equal(ev.uid) {
			return fmt.Errorf("main:config: failed to verify Unique ID (%v vs %v)", flags.UID.ToString(), ev.uid.ToString())
		}

		break

	case <-time.After(1000 * time.Millisecond):
		return fmt.Errorf("main:config: failed to request NID: %s", "timeout")
	}

	return nil
}

func configureGateway(remote Remote, flags Flags) error {
	configMode, err := remoteInConfigMode(remote)

	if nil != err {
		return err
	} else if !configMode {
		_, err = remote.Write(SetGwConfigModeCmd([]byte{0, 0, 0, 0}), -1)

		if nil != err {
			return err
		}

		configMode, err = remoteInConfigMode(remote)

		if nil != err {
			return err
		} else if !configMode {
			if !WaitForConfig(remote) {
				log.Fatalf("main:config: failed to enter config mode")
			}
		}
	}

	if err = RunConfigCmd(remote, '0', false); err != nil {
		log.Fatalf("main:config: failed to read configuration memory: %v", err)
	}

	cfg := <-remote.Channel()

	if err = RunConfigCmd(remote, 'r', false); err != nil {
		log.Fatalf("main:config: failed to read calibration memory: %v", err)
	}

	calibration := <-remote.Channel()

	usingProtocol := cfg[3]
	deviceType := cfg[14]
	uid := cfg[45:49]
	sid := cfg[49:53]
	nid := calibration[23:27]

	log.Printf("main:config: protocol=%v deviceType=%v uid=%v sid=%v nid=%v",
		usingProtocol,
		deviceType,
		AddressToString(uid),
		AddressToString(sid),
		AddressToString(nid))

	if 1 != deviceType {
		log.Println("main:config: ensure gateway operations")
		if err = RunConfigCmd(remote, 'G', true); err != nil {
			log.Fatalf("main:config: failed to enable gateway mode: %v", err)
		}
	}

	newCfg := []ConfigValue{}

	if 0 != usingProtocol {
		newCfg = append(newCfg, ConfigValue{3, 0})
	}

	if !flags.UID.Equal(uid) {
		newCfg = append(newCfg, ConfigValue{45, flags.UID[0]})
		newCfg = append(newCfg, ConfigValue{46, flags.UID[1]})
		newCfg = append(newCfg, ConfigValue{47, flags.UID[2]})
		newCfg = append(newCfg, ConfigValue{48, flags.UID[3]})
	}

	if !flags.SID.Equal(sid) {
		newCfg = append(newCfg, ConfigValue{49, flags.SID[0]})
		newCfg = append(newCfg, ConfigValue{50, flags.SID[1]})
		newCfg = append(newCfg, ConfigValue{51, flags.SID[2]})
		newCfg = append(newCfg, ConfigValue{52, flags.SID[3]})
	}

	if len(newCfg) > 0 {
		log.Println("main:config: set configuration")
		if err = SetConfigurationMemory(remote, newCfg); err != nil {
			log.Fatalf("main:config:failed to set configuration memory: %v\n :: %v\n", newCfg, err)
		}
	}

	if !flags.NID.Equal(nid) {
		setNID := []ConfigValue{
			ConfigValue{23, flags.NID[0]},
			ConfigValue{24, flags.NID[1]},
			ConfigValue{25, flags.NID[2]},
			ConfigValue{26, flags.NID[3]},
		}

		log.Println("main:config: set calibration")
		if err = SetCalibrationMemory(remote, setNID); err != nil {
			log.Fatalf("main:config:failed to set calibration memory: %v\n :: %v\n", setNID, err)
		}
	}

	// log.Println("config-mode: EXIT")
	if err = RunConfigCmd(remote, 'X', false); err != nil {
		log.Fatalf("main:config: failed to exit configuration mode: %v", err)
	}

	return nil
}

// WaitForConfig attempt to put `remote` in configuration mode
func WaitForConfig(remote Remote) bool {
	for {
		select {
		case prompt := <-remote.Channel():
			if len(prompt) == 0 || prompt[0] != '>' {
				return false
			}

			return true

		case <-time.After(500 * time.Millisecond):
			continue
		}
	}
}
