package dreamhostregistry

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/context"
)

var DefaultAPIEndpoint = "https://api.dreamhost.com"

type Registry struct {
	APIKey      string
	APIEndpoint string
	RecordName  string
	HTTPClient  *http.Client
}

func (r *Registry) http() *http.Client {
	if r.HTTPClient != nil {
		return r.HTTPClient
	}
	return http.DefaultClient
}

func (r *Registry) apiEndpoint() string {
	if r.APIEndpoint != "" {
		return r.APIEndpoint
	}
	return DefaultAPIEndpoint
}

func (r *Registry) api(ctx context.Context, cmd string, params map[string]string) (map[string]interface{}, error) {
	_params := make(url.Values, len(params)+3)
	_params.Set("format", "json")
	_params.Set("key", r.APIKey)
	_params.Set("cmd", cmd)
	for k, v := range params {
		_params.Set(k, v)
	}
	body := _params.Encode()
	req, err := http.NewRequest("POST", r.apiEndpoint(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	type result struct {
		*http.Response
		error
	}
	res := make(chan result, 1)
	client := r.http()
	go func() {
		resp, err := client.Do(req)
		res <- result{resp, err}
	}()
	select {
	case <-ctx.Done():
		type canceller interface {
			CancelRequest(*http.Request)
		}
		transport := client.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		c, ok := client.Transport.(canceller)
		if ok {
			c.CancelRequest(req)
		} else {
			log.Printf("dreamhostregistry: http transport cannot cancel requests")
		}
		return nil, fmt.Errorf("context terminated")
	case r := <-res:
		if r.error != nil {
			return nil, r.error
		}
		defer r.Response.Body.Close()
		body := io.LimitReader(r.Response.Body, 4<<10)
		var result map[string]interface{}
		err = json.NewDecoder(body).Decode(&result)
		if err != nil {
			return nil, err
		}
		if result["result"] == "error" {
			return nil, &APIError{cmd, result["data"], result["reason"]}
		}
		return result, nil
	}
}

type APIError struct {
	Cmd    string
	Data   interface{}
	Reason interface{}
}

func (e *APIError) Error() string {
	if e.Reason == nil {
		if e.Data == nil {
			return fmt.Sprintf("%s: unknown_error", e.Cmd)
		}
		return fmt.Sprintf("%s: %v", e.Cmd, e.Data)
	}
	return fmt.Sprintf("%s: %v -- %v", e.Cmd, e.Data, e.Reason)
}

func (r *Registry) SetName(ctx context.Context, ip string) error {
	record, ok := r.findExisting(ctx)
	if !ok {
		log.Printf("dreamhostregistry: record not found -- attempting to add")
		record, err := r.addRecord(ctx, ip)
		if err != nil {
			log.Printf("dreamhostregistry: %v", err)
		} else {
			log.Printf("dreamhostregistry: record added -- %q", record)
		}
		return err
	}

	oldip, _ := record["value"].(string)
	if oldip == ip {
		log.Printf("dreamhostregistry: no change on remote -- comment=%q", record["comment"])
		return nil
	}
	err := r.removeExisting(ctx, oldip)
	if err != nil {
		log.Printf("dreamhostregistry: %v", err)
	}

	_record, err := r.addRecord(ctx, ip)
	if err != nil {
		log.Printf("dreamhostregistry: %v", err)
	} else {
		log.Printf("dreamhostregistry: record updated -- %q", _record)
	}
	return err
}

func (r *Registry) addRecord(ctx context.Context, ip string) (map[string]string, error) {
	cmd := "dns-add_record"
	record := map[string]string{
		"record":  r.RecordName,
		"type":    "A",
		"value":   ip,
		"comment": fmt.Sprintf("set by pmdns at %s", time.Now().Format(time.RFC3339)),
	}
	_, err := r.api(ctx, cmd, record)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (r *Registry) removeExisting(ctx context.Context, oldip string) error {
	cmd := "dns-remove_record"
	params := map[string]string{
		"record": r.RecordName,
		"type":   "A",
		"value":  oldip,
	}
	_, err := r.api(ctx, cmd, params)
	return err
}

func (r *Registry) findExisting(ctx context.Context) (map[string]interface{}, bool) {
	result, err := r.api(ctx, "dns-list_records", nil)
	if err != nil {
		return nil, false
	}
	records, ok := result["data"].([]interface{})
	if !ok {
		return nil, false
	}
	for _, rdata := range records {
		record, _ := rdata.(map[string]interface{})
		name, _ := record["record"].(string)
		if name == r.RecordName {
			return record, true
		}
	}
	return nil, false
}
