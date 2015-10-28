package main

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"

	_ "expvar"
	_ "net/http/pprof"
)

type IPDiscoveryService interface {
	GetIP(ctx context.Context) (string, error)
}

type IPDiscovery struct {
	service IPDiscoveryService
	context context.Context
}

func (disco *IPDiscovery) DiscoverIP(tick <-chan time.Time) *IPDiscoveryPoller {
	return newIPDiscoveryPoller(disco.context, disco.service, tick)
}

type IPDiscoveryPoller struct {
	context context.Context
	service IPDiscoveryService
	tick    <-chan time.Time
	ip      chan string
	Timeout time.Duration
	Log     func(err error)
}

func newIPDiscoveryPoller(ctx context.Context, service IPDiscoveryService, tick <-chan time.Time) *IPDiscoveryPoller {
	poller := &IPDiscoveryPoller{
		context: ctx,
		service: service,
		tick:    tick,
		ip:      make(chan string),
	}
	go poller.loop()
	return poller
}

func (poller *IPDiscoveryPoller) log(err error) {
	if poller.Log != nil {
		poller.Log(err)
		return
	}
	log.Printf("discovery: %v", err)
	return
}

func (poller *IPDiscoveryPoller) loop() {
	defer close(poller.ip)

	poll := make(chan pollResult, 1)
	var pollcancel func()
	var ipchan chan<- string
	var ip string
	var iptime time.Time
	term := poller.context.Done()
	for {
		select {
		case <-term:
			if pollcancel != nil {
				pollcancel()
				pollcancel = nil
			}
			return
		case _, ok := <-poller.tick:
			if pollcancel != nil {
				pollcancel()
				pollcancel = nil
			}
			if !ok {
				return
			}

			ctx, cancel := context.WithCancel(poller.context)
			pollcancel = cancel

			go poller.poll(ctx, poll)
		case ipchan <- ip:
			ipchan = nil
		case result := <-poll:
			if result.err != nil {
				poller.log(result.err)
				continue
			}
			if result.time.After(iptime) {
				ip = result.ip
				ipchan = poller.ip
			} else {
				rtime := result.time.Format(time.RFC3339)
				last := iptime.Format(time.RFC3339)
				err := fmt.Errorf("time in the past: %v (last at %v)", rtime, last)
				poller.log(err)
			}
		}
	}
}

func (poller *IPDiscoveryPoller) poll(ctx context.Context, result chan<- pollResult) {
	timeout := poller.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	now := time.Now()
	ip, err := poller.service.GetIP(ctx)
	r := pollResult{ip, now, err}
	select {
	case <-ctx.Done():
	case result <- r:
	}
}

type pollResult struct {
	ip   string
	time time.Time
	err  error
}

func (poller *IPDiscoveryPoller) IP() <-chan string {
	return poller.ip
}
