// Package boottimer provides a timer that is accurate over suspend.
package boottimer

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// #cgo LDFLAGS: -lrt
//
// #include <signal.h>
// #include <time.h>
import "C"

var (
	Alarms = make(chan struct{})
)

// Set a timer for a specific duration.
//
// The timer will use CLOCK_BOOTTIME. When the system sleeps and wakes up, this
// clock reflects the period that it was offline, which avoids the timer
// getting "postponed" due to the system sleeping.
//
// If a timer were to expire at a time when the system is still suspended, it
// will expire as soon as the system wakes up again.
//
// Because this uses a POSIX alarm under the hood, all alarms are notified via
// the same channel `Alarms` above.
func SetTimer(d time.Duration) {
	var timer C.timer_t
	C.timer_create(C.CLOCK_BOOTTIME, nil, &timer)

	seconds := d.Round(time.Second).Seconds()
	ns := (d - d.Truncate(time.Second)).Nanoseconds()
	log.Printf("Setting timer for %v.%v seconds.\n", seconds, ns)

	var spec = C.struct_itimerspec{
		it_interval: C.struct_timespec{
			tv_sec:  0,
			tv_nsec: 0,
		},
		it_value: C.struct_timespec{
			tv_sec:  C.time_t(seconds),
			tv_nsec: C.long(ns),
		},
	}

	C.timer_settime(timer, 0, &spec, nil)
}

func init() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGALRM)

	go func() {
		for {
			s := <-c
			log.Println("Got signal:", s)
			Alarms <- struct{}{}
		}
	}()
}
