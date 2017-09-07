package guri

import (
	"log"
	"time"
)

func forward(remote Remote, t time.Duration) chan []byte {
	ch := make(chan []byte, 256)

	go func() {
		for {
			buf, err := remote.Recv(t)

			if nil != err {
				log.Printf("forward-err: %v\n", err)
				close(ch)
				return
			}

			ch <- buf
		}
	}()

	return ch
}

// Loop run "event" loop
func Loop(from Remote, to Remote, flags Flags) {
	upstream := forward(from, 500*time.Millisecond)
	downstream := forward(to, 2*time.Millisecond)

	upoff := &Backoff{
		initial: 1 * time.Second,
		wait:    1 * time.Second,
		delay:   2.5,
		max:     5 * time.Minute,
	}

	downoff := &Backoff{
		initial: 1 * time.Second,
		wait:    1 * time.Second,
		delay:   2.5,
		max:     5 * time.Minute,
	}

	for {
		select {
		case buf, state := <-upstream:
			if false == state {
				if !flags.Reconnect {
					log.Fatalf("upstream:close, exiting")
				}

				log.Printf("upstream:close, reconnecting\n")
				from.Close()
				if err := from.Connect(); nil != err {
					log.Printf("upstream:open: %v\n", err)
					upoff.Fail()
				} else {
					upoff.Success()
					upstream = forward(from, 500*time.Millisecond)
				}
			} else if len(buf) > 0 {
				log.Printf("upstream:recv[%v] %v\n", state, buf)
				if len(buf) > 10 && 6 == buf[0] {
					to.Write(buf[:1], -1)
					to.Write(buf[1:], -1)
				} else {
					to.Write(buf, -1)
				}
			}

		case buf, state := <-downstream:
			if false == state {
				if !flags.Reconnect {
					log.Fatalf("upstream:close, exiting")
				}

				to.Close()
				log.Printf("downstream:close, reconnecting\n")
				if err := to.Connect(); nil != err {
					log.Printf("downstream:open: %v\n", err)
					downoff.Fail()
				} else {
					downoff.Success()
					downstream = forward(to, 2*time.Millisecond)
				}
			} else if len(buf) > 0 {
				log.Printf("downstream:recv[%v] %v\n", state, buf)
				from.Write(buf, -1)
			}
		}
	}
}
