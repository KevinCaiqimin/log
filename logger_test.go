package log

import (
	"fmt"
	"os"
	"os/signal"
	"testing"
	"time"
)

func TestDebug(t *testing.T) {
	InitLog("aaa", "HOUR", LV_DEBUG)
	Debug("Hello")

	worker_num := 1
	for i := 0; i < worker_num; i++ {
		go func() {
			cnt := 0
			for {
				Debug("this is for teis is for teis is for teis is for teis is for teis is for test: %v=%v", cnt, cnt)
				cnt++
			}
		}()
	}

	go func() {
		for {
			time.Sleep(time.Second * 3)
			fmt.Printf("log queue len=%d, buf_len=%d\n", len(getInstance().ch), len(getInstance().buf.Bytes()))
		}
	}()

	sig_ch := make(chan os.Signal, 1)
	signal.Notify(sig_ch, os.Interrupt)
	<-sig_ch
}
