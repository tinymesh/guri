package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	guri "github.com/tinymesh/guri/guri"
)

var (
	vsn = "0.0.1-rc3"
)

func parseFlags() guri.Flags {
	flags := new(guri.Flags)

	// commands
	listFlag := flag.Bool("list", false, "List available serialports")
	helpFlag := flag.Bool("help", false, "Show help text")
	versionFlag := flag.Bool("version", false, "Show version")

	// link flags
	verifyFlag := flag.Bool("verify", true, "validate IDs according to -nid, -sid, and -uid flags")
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

	flags.Help = *helpFlag
	flags.List = *listFlag
	flags.Version = *versionFlag

	flags.Verify = *verifyFlag
	flags.AutoConfigure = *autoConfigureFlag
	flags.NID = guri.ParseAddr(*nidFlag)
	flags.SID = guri.ParseAddr(*sidFlag)
	flags.UID = guri.ParseAddr(*uidFlag)

	if len(flags.NID) == 0 {
		log.Fatalf("failed to parse -nid value, value must be 4 bytes encoded as hexadecimals with : as a separator\nexample: -nid 01:02:03:04\n")
	}
	if len(flags.SID) == 0 {
		log.Fatalf("failed to parse -sid value, value must be 4 bytes encoded as hexadecimals with : as a separator\nexample: -sid 01:02:03:04\n")
	}
	if len(flags.UID) == 0 {
		log.Fatalf("failed to parse -uid value, value must be 4 bytes encoded as hexadecimals with : as a separator\nexample: -uid 01:02:03:04\n")
	}

	flags.Stdio = *stdioFlag
	flags.Remote = *remoteFlag
	flags.TLS = *usetlsFlag
	flags.Reconnect = *reconnectFlag

	return *flags
}

func pickUpstream(flags guri.Flags) (guri.Remote, error) {
	if true == flags.Stdio {
		// stdio
		return guri.ConnectStdio(os.Stdin, os.Stdout)
	} else if true == flags.TLS {
		// tls
		return guri.ConnectTLS(flags.Remote)
	}

	return guri.ConnectTCP(flags.Remote)
}

func main() {

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	flags := parseFlags()

	if true == flags.Help {
		flag.PrintDefaults()
		return
	} else if true == flags.List {
		guri.PrintPortList()
		return
	} else if true == flags.Version {
		fmt.Printf("%v\n", vsn)
		return
	}

	path := flag.Arg(0)

	if "" == path {
		log.Fatal(errors.New("1st argument, tty path, missing"))
	}

	var upstream guri.Remote
	var downstream guri.Remote
	var err error

	log.Printf("guri - version %v\n", vsn)

	if downstream, err = guri.ConnectSerial(path, flags); nil != err {
		log.Fatal(err)
	}

	if upstream, err = pickUpstream(flags); nil != err {
		log.Fatalf("failed to connect to upstream; %v\n", err)
	} else {
		guri.Loop(upstream, downstream, flags)
	}
}
