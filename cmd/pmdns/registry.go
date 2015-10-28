package main

import (
	"log"
	"time"

	"golang.org/x/net/context"
)

type NameRegistry interface {
	SetName(ctx context.Context, ip string) error
}

type Registrar struct {
	context  context.Context
	registry NameRegistry
}

func (r *Registrar) RegisterIP(ip <-chan string, cachettl time.Duration) <-chan struct{} {
	if cachettl <= 0 {
		cachettl = time.Hour
	}
	m := &registrationManager{
		context:  r.context,
		registry: r.registry,
		ip:       ip,
		CacheTTL: cachettl,
	}
	done := make(chan struct{})
	go m.loop(done)
	return done
}

type registrationManager struct {
	context  context.Context
	registry NameRegistry
	ip       <-chan string
	CacheTTL time.Duration
}

func (m *registrationManager) loop(done chan<- struct{}) {
	defer close(done)

	var lastIP string
	var lastUpdate time.Time
	updateCancel := func() {}

	defer updateCancel()

	for {
		select {
		case newIP, ok := <-m.ip:
			if !ok {
				return
			}

			if lastIP != newIP {
				log.Printf("registration: new ip -- new=%s old=%s lastUpdate=%s", newIP, lastIP, lastUpdate.Format(time.RFC3339))
			} else {
				cacheok := time.Since(lastUpdate) < m.CacheTTL
				log.Printf("registration: no change -- ip=%s cacheok=%t", newIP, cacheok)
				if cacheok {
					continue
				}
			}

			lastIP = newIP
			lastUpdate = time.Now()

			if updateCancel != nil {
				updateCancel()
			}
			ctx, cancel := context.WithTimeout(m.context, 30*time.Second)
			updateCancel = cancel
			go func() {
				defer cancel()
				m.registry.SetName(ctx, newIP)
			}()
		case <-m.context.Done():
			return
		}
	}
}
