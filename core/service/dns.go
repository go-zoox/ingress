package service

import (
	"fmt"
	"net"
)

func (s *Service) CheckDNS() (ips []string, err error) {
	if s.Name == "" {
		return nil, fmt.Errorf("service name is required")
	}

	// client := dns.NewClient()

	// ips, err = client.LookUp(name)
	// if err != nil {
	// 	// Not Found
	// 	if err.Error() == "failed to query with code: 3" {
	// 		return nil, fmt.Errorf("service %s not found", name)
	// 	}

	// 	return nil, err
	// }

	// if len(ips) == 0 {
	// 	return nil, fmt.Errorf("service %s not found with 0 ips", name)
	// }

	return net.LookupHost(s.Name)
}
