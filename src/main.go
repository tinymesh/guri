package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

var (
	vsn = "0.0.1-rc2"
)

type Remote interface {
	Channel() chan []byte
	Recv(timeout time.Duration) ([]byte, error)
	Write(buf []byte, timeout time.Duration) (int, error)
	Open() chan []byte
	Close() error
}

type Flags struct {
	help    bool
	list    bool
	version bool

	verify        bool
	nid           Address
	sid           Address
	uid           Address
	autoConfigure bool

	stdio     bool
	remote    string
	tls       bool
	reconnect bool
}

func parseFlags() Flags {
	flags := new(Flags)

	// commands
	listFlag := flag.Bool("list", false, "List available serialports")
	helpFlag := flag.Bool("help", false, "Show help text")
	versionFlag := flag.Bool("version", false, "Show version")

	// link flags
	verifyFlag := flag.Bool("verify", false, "validate IDs according to -nid, -sid, and -uid flags")
	autoConfigureFlag := flag.Bool("auto-configure", false, "Automatically configure gateway operation and ID's; use -nid, -sid, and -uid flags")
	nidFlag := flag.String("nid", "::", "32bit Network ID in hexadecimal (ie, aa:bb:cc:dd)")
	sidFlag := flag.String("sid", "::", "32bit System ID in hexadecimal (ie, aa:bb:cc:dd)")
	uidFlag := flag.String("uid", "::", "32bit Unique ID in hexadecimal (ie, aa:bb:cc:dd)")

	// communication flags
	stdioFlag := flag.Bool("stdio", false, "Use stdio for communication instead of remote")
	remoteFlag := flag.String("remote", "tcp.cloud.tiny-mesh.com:7002", "The upstream url to connect to")
	usetlsFlag := flag.Bool("tls", true, "Controll use of TLS with -remote")
	reconnectFlag := flag.Bool("reconnect", true, "Automatically re-establish communication on failure")

	flag.Parse()

	flags.help = *helpFlag
	flags.list = *listFlag
	flags.version = *versionFlag

	flags.verify = *verifyFlag
	flags.autoConfigure = *autoConfigureFlag
	flags.nid = parseAddr(*nidFlag)
	flags.sid = parseAddr(*sidFlag)
	flags.uid = parseAddr(*uidFlag)

	if len(flags.nid) == 0 {
		log.Fatalf("failed to parse -nid value, value must be 4 bytes encoded as hexadecimals with : as a separator\nexample: -nid 01:02:03:04\n")
	}
	if len(flags.sid) == 0 {
		log.Fatalf("failed to parse -sid value, value must be 4 bytes encoded as hexadecimals with : as a separator\nexample: -sid 01:02:03:04\n")
	}
	if len(flags.uid) == 0 {
		log.Fatalf("failed to parse -uid value, value must be 4 bytes encoded as hexadecimals with : as a separator\nexample: -uid 01:02:03:04\n")
	}

	flags.stdio = *stdioFlag
	flags.remote = *remoteFlag
	flags.tls = *usetlsFlag
	flags.reconnect = *reconnectFlag

	return *flags
}

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
		return fmt.Errorf("main:config: Device in config mode... you must exit manually\n")
	}
	_, err = remote.Write(GetNIDCmd([]byte{0, 0, 0, 0}), -1)

	select {
	case nidEv := <-remote.Channel():
		ev, err := decode(nidEv)

		if err != nil {
			log.Fatal(err)
		}

		if !flags.nid.Equal(ev.address) {
			return fmt.Errorf("main:config: failed to verify Network ID (%v vs %v)", flags.nid.ToString(), ev.address.ToString())
		} else if !flags.sid.Equal(ev.sid) {
			return fmt.Errorf("main:config: failed to verify System ID (%v vs %v)", flags.sid.ToString(), ev.sid.ToString())
		} else if !flags.uid.Equal(ev.uid) {
			return fmt.Errorf("main:config: failed to verify Unique ID (%v vs %v)", flags.uid.ToString(), ev.uid.ToString())
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

		configMode, err := remoteInConfigMode(remote)

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
		log.Println("main:config: set configuration")
		if err = SetConfigurationMemory(remote, newCfg); err != nil {
			log.Fatalf("main:config:failed to set configuration memory: %v\n :: %v\n", newCfg, err)
		}
	}

	if !flags.nid.Equal(nid) {
		setNid := []ConfigValue{
			ConfigValue{23, flags.nid[0]},
			ConfigValue{24, flags.nid[1]},
			ConfigValue{25, flags.nid[2]},
			ConfigValue{26, flags.nid[3]},
		}

		log.Println("main:config: CMD: calibration")
		if err = SetCalibrationMemory(remote, setNid); err != nil {
			log.Fatalf("main:config:failed to set calibration memory: %v\n :: %v\n", setNid, err)
		}
	}

	// log.Println("config-mode: EXIT")
	if err = RunConfigCmd(remote, 'X', false); err != nil {
		log.Fatalf("main:config: failed to exit configuration mode: %v", err)
	}

	return nil
}

func WaitForConfig(remote Remote) bool {
	for {
		select {
		case prompt := <-remote.Channel():
			if len(prompt) == 0 || prompt[0] != '>' {
				return false
			} else {
				return true
			}

		case <-time.After(500 * time.Millisecond):
			continue
		}
	}
}

func main() {

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	flags := parseFlags()

	if true == flags.help {
		flag.PrintDefaults()
		return
	} else if true == flags.list {
		PrintPortList()
		return
	} else if true == flags.version {
		fmt.Printf("%v\n", vsn)
		return
	}

	path := flag.Arg(0)

	if "" == path {
		log.Fatal(errors.New("1st argument, tty path, missing"))
	}

	var upstream Remote
	var downstream Remote
	var err error

	log.Printf("guri - version %v\n", vsn)
	downstream, err = ConnectSerial(path)

	if nil != err {
		log.Fatal(err)
	}

	downstreamchan := downstream.Open()

	if flags.verify && !flags.autoConfigure {
		err := verifyIDs(downstream, flags)

		if nil != err {
			log.Fatal(err)
		}
	} else if flags.autoConfigure {
		err := configureGateway(downstream, flags)

		if nil != err {
			log.Fatal(err)
		}

		err = verifyIDs(downstream, flags)
		if nil != err {
			log.Fatal(err)
		}
	}

	connectUpstream := func() (Remote, error) {

		if true == flags.stdio {
			upstream, err = ConnectStdio(os.Stdin, os.Stdout)

			if nil != err {
				log.Fatalf("error[stdio] %v\n", err)
			}
		} else if true == flags.tls {
			// setup remote TLS communication
			upstream, err = ConnectTLS(flags.remote)

			if nil != err {
				log.Printf("error[tcp/tls] %v\n", err)
				return nil, err
			}
		} else {
			// setup remote TCP communication without TLS
			upstream, err = ConnectTCP(flags.remote)

			if nil != err {
				log.Printf("error[tcp] %v\n", err)
				return nil, err
			}
		}

		if nil == upstream {
			return nil, errors.New("no upstream configured")
		}

		return upstream, nil
	}

	upstream, err = connectUpstream()

	// downstreamchan := downstream.Open()
	upstreamchan := upstream.Open()

	var maxRetries = 0

	if true == flags.reconnect {
		maxRetries = -1
	}

	var upstreamBackoff Backoff = NewBackoff(time.Second, 2.0, maxRetries)
	var downstreamBackoff Backoff = NewBackoff(time.Second, 2.0, maxRetries)

	for {
		select {
		case buf, state := <-downstreamchan:
			if false == state {
				if true == flags.reconnect {
					log.Printf("downstream:close, reconnecting\n")
					downstreamBackoff.Until(func() error {
						downstream, err = ConnectSerial(path)
						if nil != err {
							log.Printf("error[downstream:connect] %v\n", err)
						} else {
							log.Printf("downstream:connect reconnected\n")
						}
						return err
					})

					downstreamchan = downstream.Open()
				} else {
					log.Printf("downstream:close, terminating\n")
					return
				}
			} else {
				log.Printf("downstream:recv[%v] %v\n", state, buf)
				upstream.Write(buf, -1)
			}

		case buf, state := <-upstreamchan:
			if false == state {
				if true == flags.reconnect {
					log.Printf("upstream:close, reconnecting\n")
					upstreamBackoff.Until(func() error {
						upstream, err = connectUpstream()
						if nil != err {
							log.Printf("error[upstream:connect] %v\n", err)
						} else {
							log.Printf("upstream:connect reconnect\n")
						}
						return err
					})

					upstreamchan = upstream.Open()
				} else {
					log.Printf("upstream:close, terminating\n")
					return
				}

			} else {
				log.Printf("upstream:recv[%v] %v\n", state, buf)
				if true == flags.stdio && buf[0] == 10 {
					downstream.Write([]byte("\x0a\x00\x00\x00\x00\x03\x03\x10\x00\x00"), -1)
				}

				downstream.Write(buf, -1)
			}
		}
	}
}
