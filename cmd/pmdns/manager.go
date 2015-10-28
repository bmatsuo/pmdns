package main

import (
	"sync"
	"time"

	"golang.org/x/net/context"
)

var DefaultDynamicIPOptions = &DynamicIPOptions{
	Context:      context.Background(),
	PollInterval: 5 * time.Minute,
	NameCacheTTL: time.Hour,
}

type DynamicIPManager struct {
	context   context.Context
	options   *DynamicIPOptions
	disco     *IPDiscovery
	reg       *Registrar
	term      func()
	pipedone  chan struct{}
	statsLock sync.RWMutex
	stats     DynamicIPStats
}

func (m *DynamicIPManager) Stats() *DynamicIPStats {
	s := &DynamicIPStats{}
	*s = m.stats
	return s
}

type DynamicIPStats struct {
	LastPoll   string
	LastPollNs int64
}

type DynamicIPOptions struct {
	Context      context.Context
	PollInterval time.Duration
	NameCacheTTL time.Duration
}

func NewDynamicIPOptions() *DynamicIPOptions {
	opt := &DynamicIPOptions{}
	*opt = *DefaultDynamicIPOptions
	return opt
}

func NewDynamicIPManager(disco IPDiscoveryService, reg NameRegistry, opt *DynamicIPOptions) *DynamicIPManager {
	if opt == nil {
		opt = NewDynamicIPOptions()
	}
	ctx, term := context.WithCancel(opt.Context)
	m := &DynamicIPManager{
		context: ctx,
		options: opt,
		disco: &IPDiscovery{
			context: ctx,
			service: disco,
		},
		reg: &Registrar{
			context:  ctx,
			registry: reg,
		},
		term:     term,
		pipedone: make(chan struct{}),
	}
	go m.run()
	return m
}

func (m *DynamicIPManager) Wait() <-chan struct{} {
	return m.pipedone
}

func (m *DynamicIPManager) run() {
	defer close(m.pipedone)
	tick := make(chan time.Time, 1)
	tick <- time.Now()
	tickDisco := time.NewTicker(m.options.PollInterval)
	go func() {
		for {
			select {
			case t := <-tickDisco.C:
				select {
				case tick <- t:

				default:
				}
			case <-m.context.Done():
				tickDisco.Stop()
				return
			}
		}
	}()
	ipchan := make(chan string)
	updateStats := func() {
		m.statsLock.Lock()
		defer m.statsLock.Unlock()
		utcnow := time.Now().UTC()
		m.stats.LastPollNs = utcnow.UnixNano()
		m.stats.LastPoll = utcnow.Format(time.RFC3339Nano)
	}
	go func() {
		for ip := range m.disco.DiscoverIP(tick).IP() {
			ipchan <- ip
			updateStats()
		}
	}()
	done := m.reg.RegisterIP(ipchan, m.options.NameCacheTTL)
	<-done
}
