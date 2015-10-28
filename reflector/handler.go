package reflector

import (
	"fmt"
	"log"
	"net"
	"net/http"
)

var DefaultAddrHandler = func(w http.ResponseWriter, r *http.Request, addr *RemoteAddr) {
	fmt.Fprintln(w, addr.Host)
}

var DefaultErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
	DefaultErrorLogger(err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

var DefaultErrorLogger = func(err error) {
	log.Printf("reflector: %v", err)
}

type Handler struct {
	HandleAddr  func(w http.ResponseWriter, r *http.Request, addr *RemoteAddr)
	LogError    func(err error)
	HandleError func(w http.ResponseWriter, r *http.Request, err error)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body.Close()

	addr, err := ParseRemoteAddr(r.RemoteAddr)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	h.handleAddr(w, r, addr)
}

func (h *Handler) handleAddr(w http.ResponseWriter, r *http.Request, addr *RemoteAddr) {
	if h.HandleAddr != nil {
		h.handleAddr(w, r, addr)
		return
	}
	DefaultAddrHandler(w, r, addr)
}

func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if h.HandleError != nil {
		h.HandleError(w, r, err)
		return
	}
	DefaultErrorHandler(w, r, err)
}

func ParseRemoteAddr(addr string) (*RemoteAddr, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	raddr := &RemoteAddr{
		HostPort: addr,
		Host:     host,
		Port:     port,
	}
	return raddr, nil
}

type RemoteAddr struct {
	HostPort string
	Host     string
	Port     string
}
