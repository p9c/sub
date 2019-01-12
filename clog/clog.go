package clog

// Miscellaneous functions

import (
	"fmt"
	"time"

	"github.com/logrusorgru/aurora"
)

// Check checks if an error exists, if so, prints a log to the specified log level with a string and returns if err was nil
func Check(err error, tag int, where string) (wasError bool) {
	if err != nil {
		L[tag].Chan <- L[tag].Name + " " + err.Error()
		if tag == Ftl.Num {
			panic("died")
		}
	}
	return
}

// Lvl is a log level data structure
type Lvl struct {
	Num  int
	Name string
	Chan chan string
}

var (
	// Ftl is for critical/fatal errors
	Ftl = Lvl{0, aurora.BgRed("FTL").String(), nil}
	// Err is an error that does block continuation
	Err = Lvl{1, aurora.Red("ERR").String(), nil}
	// Wrn is is a warning of a correctable condition
	Wrn = Lvl{2, aurora.Brown("WRN").String(), nil}
	// Inf is is general information
	Inf = Lvl{3, aurora.Green("INF").String(), nil}
	// Dbg is debug level information
	Dbg = Lvl{4, aurora.Blue("DBG").String(), nil}
	// Trc is detailed outputs of contents of variables
	Trc = Lvl{5, aurora.BgBlue("TRC").String(), nil}
)

// L is an array of log levels that can be selected given the level number
var L = []Lvl{
	Ftl,
	Err,
	Wrn,
	Inf,
	Dbg,
	Trc,
}

// Logger is a short access method
type Logger struct {
	Ftl   chan string
	Err   chan string
	Wrn   chan string
	Inf   chan string
	Dbg   chan string
	Trc   chan string
	Start func(...func(name, txt string))
	Stop  func()
}

// Get returns a structure with the struct fields for each loglevel
func Get(fn ...func(...func(name, txt string))) (rl Logger, rls []Lvl) {
	rl.Start = Init
	if fn != nil {
		rl.Start = fn[0]
	}
	return Logger{
			Ftl.Chan,
			Err.Chan,
			Wrn.Chan,
			Inf.Chan,
			Dbg.Chan,
			Trc.Chan,
			rl.Start,
			func() {
				close(Quit)
			},
		},
		L
}

// LogLevel is a dynamically settable log level filter that excludes higher values from output
var LogLevel = Trc.Num

// Quit signals the logger to stop
var Quit = make(chan struct{})

// LogIt is the function that performs the output, can be loaded by the caller
var LogIt = Print

// Init manually starts a clog
func Init(fn ...func(name, txt string)) {
	var ready []chan bool
	// override the output function if one is given
	if len(fn) > 0 {
		LogIt = fn[0]
	}
	for range L {
		ready = append(ready, make(chan bool))
	}
	for i := range L {
		go startChan(i, ready[i])
	}
	for i := range ready {
		<-ready[i]
	}
	Print("logger started", "")
}

// Print out a formatted log message
func Print(name, txt string) {
	fmt.Printf("%s [%s] %s\n",
		time.Now().UTC().Format("2006-01-02 15:04:05.000000 MST"),
		name,
		txt,
	)
}

func startChan(ch int, ready chan bool) {
	L[ch].Chan = make(chan string)
	ready <- true
	done := true
	for done {
		select {
		case <-Quit:
			done = false
			continue
		case txt := <-L[ch].Chan:
			if ch <= LogLevel {
				LogIt(L[ch].Name, txt)
			}
			continue
		default:
		}
	}
}
