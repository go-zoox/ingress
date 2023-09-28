package core

import (
	"fmt"

	"github.com/go-zoox/dns"
)

func (c *core) CheckDNS(name string) (ok bool, ips []string, err error) {
	client := dns.NewClient()

	if name == "" {
		return false, nil, fmt.Errorf("service name is empty")
	}

	ips, err = client.LookUp(name)
	if err != nil {
		// Not Found
		// if errors.Is(err, miekg.RcodeNameError) {
		// 	return false, nil
		// }
		if err.Error() == "failed to query with code: 3" {
			return false, nil, nil
		}

		return false, nil, err
	}

	if len(ips) == 0 {
		return false, nil, fmt.Errorf("service %s not found with 0 ips", name)
	}

	return true, ips, nil
}
