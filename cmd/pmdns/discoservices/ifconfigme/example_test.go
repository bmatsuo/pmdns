package ifconfigme_test

import (
	"fmt"
	"net"
	"net/http"
	"time"

	ifconfigme "."
	"golang.org/x/net/context"
)

func Example() {
	s := new(ifconfigme.Service)

	s.HTTPClient = &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{Timeout: 5 * time.Second}).Dial,
		},
	}

	c := context.Background()

	c, cancel := context.WithTimeout(c, 5*time.Second)
	defer cancel()

	ip, err := s.GetIP(c)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(ip)
	// Output: foo
}
