package dreamhostregistry

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/context"
)

func Example() {
	r := &Registry{
		APIKey:     os.Getenv("DREAMHOST_API_KEY"),
		RecordName: os.Getenv("DREAMHOST_RECORD_NAME"),
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{Timeout: 5 * time.Second}).Dial,
			},
		},
	}

	c := context.Background()
	c, cancel := context.WithTimeout(c, 10*time.Second)
	defer cancel()

	err := r.SetName(c, os.Getenv("DREAMHOST_RECORD_VALUE"))
	if err != nil {
		fmt.Println(err)
	}
	// Output:
}
