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
	// for alarm events. This makes sure that starting the listener is
	// atomic, so we don't end up with zombie goroutines.
	alarmSignalLock sync.Mutex
	c               chan struct{}
	listenerRunning = false
)

func listenToAlarm(c chan struct{}) {
	alarmSignalLock.Lock()
	if listenerRunning {
		return
	}
	listenerRunning = true
	alarmSignalLock.Unlock()

	c2 := make(chan os.Signal, 1)
	signal.Notify(c2, syscall.SIGALRM)

	for {
		s := <-c2
		log.Println("Got signal:", s)
		c <- struct{}{}
	}
}

// Set a timer for a specific duration.
//
// The timer will use CLOCK_BOOTTIME. When the system sleeps and wakes up, this
// clock reflects the period that it was offline, which avoids the timer
// getting "postponed" due to the system sleeping.
//
// Because this uses a POSIX alarm under the hook, there can only be one event
// listener per timer, so the channel that's past on the last call to this
// method will receive all future timer expirations.
func SetTimer(d time.Duration, c chan struct{}) {
	go listenToAlarm(c)

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
