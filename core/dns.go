package core

import (
	"fmt"

	"github.com/go-zoox/dns"
)

func (c *core) CheckDNS(name string) (ips []string, err error) {
	client := dns.NewClient()

	if name == "" {
		return nil, fmt.Errorf("service name is required")
	}

	ips, err = client.LookUp(name)
	if err != nil {
		// Not Found
		if err.Error() == "failed to query with code: 3" {
			return nil, fmt.Errorf("service %s not found", name)
		}

		return nil, err
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("service %s not found with 0 ips", name)
	}

	return ips, nil
}
