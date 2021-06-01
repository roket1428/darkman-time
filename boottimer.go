package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// #cgo LDFLAGS: -lrt
//
// #include <signal.h>
// #include <time.h>
import "C"

var (
	// Due to the nature of these signals, there can only be one listener
	// for alarm events.
	alarmSignalLock sync.Mutex
)

func ConnectTimers(c chan struct{}) {
	alarmSignalLock.Lock()

	c2 := make(chan os.Signal, 1)
	signal.Notify(c2, syscall.SIGALRM)

	for {
		s := <-c2
		log.Println("Got signal:", s)
		c <- struct{}{}
	}
}

func SetTimer(d time.Duration) {
	var timer C.timer_t
	C.timer_create(C.CLOCK_BOOTTIME, nil, &timer)

	seconds := d.Round(time.Second).Seconds()
	log.Printf("Setting timer for %v seconds.\n", seconds)

	var spec = C.struct_itimerspec{
		it_interval: C.struct_timespec{
			tv_sec:  0,
			tv_nsec: 0,
		},
		it_value: C.struct_timespec{
			tv_sec:  C.long(seconds),
			tv_nsec: 0,
		},
	}

	C.timer_settime(timer, 0, &spec, nil)
}
