package service

import (
	"strings"

	"github.com/go-zoox/proxy/utils/rewriter"
)

func (s *Service) Rewrite(originPath string) (newPath string) {
	r := s.rewriter()
	// if len(r) == 0 {
	// 	return originPath
	// }

	return r.Rewrite(originPath)
}

func (s *Service) rewriter() (r rewriter.Rewriters) {
	for _, rewrite := range s.Request.Path.Rewrites {
		ft := strings.Split(rewrite, ":")
		if len(ft) != 2 {
			continue
		}

		r = append(r, rewriter.Rewriter{
			From: ft[0],
			To:   ft[1],
		})
	}

	return r
}
