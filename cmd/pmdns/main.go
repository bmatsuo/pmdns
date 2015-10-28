package main

import (
	"expvar"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bmatsuo/pmdns/cmd/pmdns/discoservices/ifconfigme"
	"github.com/bmatsuo/pmdns/cmd/pmdns/registryservices/dreamhostregistry"
	"github.com/facebookgo/flagenv"
	"golang.org/x/net/context"
)

func main() {
	configLocation := flag.String("config", "/etc/pmdns/config.toml", "config location (S3 location is acceptable)")
	httpAddr := flag.String("debug.http.addr", ":9191", "debug http server bind address")
	log.Printf("ignoring config location: %v", *configLocation)

	flagenv.Prefix = "PMDNS_"
	flagenv.Parse()

	statsLock := &sync.RWMutex{}
	stats := &DynamicIPStats{}
	expfn := expvar.Func(func() interface{} {
		statsLock.RLock()
		defer statsLock.RUnlock()
		_stats := *stats
		return &_stats
	})
	expvar.Publish("DynamicIPStats", expfn)

	if *httpAddr != "" {
		go http.ListenAndServe(*httpAddr, nil)
	}

	disco := &ifconfigme.Service{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).Dial,
			},
			Timeout: 10 * time.Second,
		},
	}
	registry := &dreamhostregistry.Registry{
		APIKey:      os.Getenv("DREAMHOST_API_KEY"),
		APIEndpoint: os.Getenv("DREAMHOST_API_ENDPOINT"),
		RecordName:  os.Getenv("DREAMHOST_RECORD_NAME"),
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).Dial,
			},
			Timeout: 10 * time.Second,
		},
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	ctxRoot := context.Background()
	ctxRoot, killAll := context.WithCancel(ctxRoot)
	defer killAll()

	opt := NewDynamicIPOptions()
	opt.Context = ctxRoot
	opt.PollInterval = 30 * time.Second
	m := NewDynamicIPManager(disco, registry, opt)

	go func() {
		tick := time.NewTicker(731 * time.Millisecond)
		for {
			select {
			case <-ctxRoot.Done():
				tick.Stop()
				return
			case <-tick.C:
				_stats := m.Stats()
				statsLock.Lock()
				stats = _stats
				statsLock.Unlock()
			}
		}
	}()

	terminating := false
	for {
		select {
		case <-m.Wait():
			log.Panicf("pipeline terminated normally -- show me the stacks")
		case s := <-sig:
			if s == syscall.SIGTERM {
				log.Panicf("%s signal received", s)
			}
			if terminating {
				log.Panicf("%s signal received -- forcefully shutting down", s)
			}
			log.Printf("%s signal received -- attempting graceful shut down", s)
			terminating = true
			go killAll()
		}
	}
}
