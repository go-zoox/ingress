package service

import (
	"fmt"
)

func (s *Service) Host() string {
	if s.Port == 0 {
		s.Port = 80
	}

	proto := s.Protocol
	if proto == "" {
		proto = "http"
	}

	if proto == "http" && s.Port == 80 {
		return s.Name
	}
	if proto == "https" && s.Port == 443 {
		return s.Name
	}

	return fmt.Sprintf("%s:%d", s.Name, s.Port)
}
