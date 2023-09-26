package service

import "fmt"

func (s *Service) URLHost() string {
	if s.Port == 0 {
		s.Port = 80
	}

	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
