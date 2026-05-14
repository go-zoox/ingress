package service

import (
	"fmt"
	"strings"
)

func (s *Service) Host() string {
	proto := strings.ToLower(strings.TrimSpace(s.Protocol))
	if proto == "" {
		proto = "http"
	}

	if s.Port == 0 {
		if proto == "https" {
			s.Port = 443
		} else {
			s.Port = 80
		}
	}

	if proto == "http" && s.Port == 80 {
		return s.Name
	}
	if proto == "https" && s.Port == 443 {
		return s.Name
	}

	return fmt.Sprintf("%s:%d", s.Name, s.Port)
}
