package main

import (
	"net/http"

	"github.com/bmatsuo/pmdns/reflector"
)

func main() {
	refl := &reflector.Handler{}

	prefix := "/reflector"
	apiversion := "v1"
	apiprefix := prefix + "/api/" + apiversion
	http.Handle(apiprefix+"/ip", refl)
	http.ListenAndServe(":8080", nil)
}
