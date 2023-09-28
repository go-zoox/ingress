package service

import "fmt"

func (s *Service) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("service name is required")
	}

	return nil
}
