package sub

// Miscellaneous functions

import (
	"log"
)

func check(err error, where string, kill bool) {
	if err != nil {
		if !kill {
			log.Printf("ERR %s %s", where, err)
		} else {
			panic("FATAL " + where + " " + err.Error())
		}
	}
}
