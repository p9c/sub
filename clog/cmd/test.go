package main

import (
	"time"

	"github.com/parallelcointeam/sub/clog"
)

var l, ls = clog.Get()

func main() {
	l.Start()
	defer l.Stop()
	done := make(chan bool)
	clog.LogLevel = clog.Trc.Num
	go tests(done)
	<-done
}

func tests(done chan bool) {
	for i := range ls {
		txt := "testing log"
		ls[i].Chan <- txt
	}
	time.Sleep(time.Millisecond)
	done <- true
}
