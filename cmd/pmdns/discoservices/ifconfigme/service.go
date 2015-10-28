package ifconfigme

import (
	"net/http"

	"github.com/bmatsuo/pmdns/cmd/pmdns/discoservices/httpservice"
	"golang.org/x/net/context"
)

type Service struct {
	service    *httpservice.Service
	HTTPClient *http.Client
}

func (s *Service) GetIP(ctx context.Context) (string, error) {
	if s.service == nil {
		s.service = &httpservice.Service{
			URL:        "http://ifconfig.me/ip",
			HTTPClient: s.HTTPClient,
		}
	}
	return s.service.GetIP(ctx)
}
