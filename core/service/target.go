package service

import "fmt"

func (s *Service) Target() string {
	if s.Protocol == "" {
		s.Protocol = "http"
	}

	return fmt.Sprintf("%s://%s", s.Protocol, s.URLHost())
}
