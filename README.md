# GURI - Tinymesh Serial Adapter

GURI provides a platform independent solution for communicating with serialports.
This allows high-level software (Network Connectors) to only build the GUI part
while this application can take care of relaying data between the serialport and
a remote.

Currently a remote can be STDIO, TCP endpoint or TLS endpoint.

## Usage

```
# build your target binary
make dist/guri-linux-amd64 && dist/guri-linux-amd64 /dev/ttyUSB0
```

## Building

Build using `go build`

```
GOARCH=$TARGETARCH GOOS=$TARGETOS go build -o dist/guri-$TARGETARCH-$TARGETOS src/*.go
```

Where `$TARGETARCH` and `$TARGETOS` may be any arch and os suppported by go
