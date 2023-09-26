package service

import "fmt"

func (s *Service) Host() string {
	if s.Port == 0 {
		s.Port = 80
	}

	return fmt.Sprintf("%s:%d", s.Name, s.Port)
}
