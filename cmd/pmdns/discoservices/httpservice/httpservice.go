package httpservice

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"
)

type Service struct {
	HTTPClient  *http.Client
	MaxAttempts int
	URL         string
	BodyMaxSize int64
	ParseError  func([]byte) error
	ParseBody   func([]byte) (string, error)
}

func (s *Service) client() *http.Client {
	if s.HTTPClient == nil {
		return http.DefaultClient
	}
	return s.HTTPClient
}

func (s *Service) do(ctx context.Context, r *http.Request) (*http.Response, error) {
	client := s.client()

	done := make(chan struct{})
	defer close(done)

	type httpResult struct {
		resp *http.Response
		err  error
	}
	ch := make(chan httpResult, 1)
	go func() {
		attempts := 0
		delay := time.Millisecond
		maxdelay := time.Second
		var resp *http.Response
		var err error
	retryloop:
		for attempts < s.MaxAttempts || s.MaxAttempts <= 0 {
			attempts++
			log.Printf("httpservice: %s %s (attempt %d)", r.Method, r.URL, attempts)
			resp, err = client.Do(r)
			if err == nil {
				break
			}
			log.Printf("httpservice: %v", err)

			sleep, delay := delay, 2*delay
			if delay > maxdelay {
				delay = maxdelay
			}

			select {
			case <-time.After(sleep):
				continue
			case <-ctx.Done():
				err = fmt.Errorf("context terminated")
				break retryloop
			}
		}
		ch <- httpResult{resp, err}
	}()

	select {
	case res := <-ch:
		return res.resp, res.err
	case <-ctx.Done():
		type canceller interface {
			CancelRequest(req *http.Request)
		}
		c, ok := client.Transport.(canceller)
		if ok {
			c.CancelRequest(r)
		} else {
			log.Printf("httpservice: http transport does not support requset cancellation")
		}
		return nil, fmt.Errorf("ifconfigme: terminated")
	}
}

func (s *Service) GetIP(ctx context.Context) (string, error) {
	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.do(ctx, req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	maxsize := atomic.LoadInt64(&s.BodyMaxSize)
	if maxsize == 0 {
		maxsize = 100 << 10
	}
	body := io.LimitReader(resp.Body, maxsize)
	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		return "", err
	}

	status := resp.StatusCode
	if status == http.StatusOK {
		return s.parseBody(bytes)
	}

	err = fmt.Errorf("%s: %s", resp.Status, s.parseError(bytes))
	return "", err
}

func (s *Service) parseBody(body []byte) (string, error) {
	if s.ParseBody != nil {
		return s.ParseBody(body)
	}
	return strings.TrimSpace(string(body)), nil
}

func (s *Service) parseError(body []byte) error {
	if s.ParseError != nil {
		return s.ParseError(body)
	}
	return errors.New(string(body))
}
