package core

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/go-zoox/core-utils/strings"

	"golang.org/x/sync/errgroup"
)

func (c *core) Run() error {
	if err := c.build(); err != nil {
		return err
	}

	g := &errgroup.Group{}

	g.Go(func() error {
		if err := c.serveHTTP(); err != nil {
			c.app.Logger.Errorf("failed to start http server: %v", err)
			return err
		}
		return nil
	})

	g.Go(func() error {
		if err := c.serveHTTPs(); err != nil {
			c.app.Logger.Errorf("failed to start https server: %v", err)
			return err
		}
		return nil
	})

	return g.Wait()
}

func (c *core) serveHTTP() error {
	port := c.cfg.Port
	if port == 0 {
		port = 8080
	}

	c.app.Logger.Info("ingress start at http://127.0.0.1:%d", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), c.app)
}

func (c *core) serveHTTPs() error {
	if c.cfg.HTTPSPort == 0 {
		return nil
	}

	if len(c.cfg.SSL) == 0 {
		return fmt.Errorf("cfg.SSL is required (domain + cert)")
	}
	for _, ssl := range c.cfg.SSL {
		if ssl.Domain == "" {
			return fmt.Errorf("cfg.SSL.Domain is required")
		}
		if ssl.Cert.Certificate == "" {
			return fmt.Errorf("cfg.SSL.Cert.Certificate is required")
		}
		if ssl.Cert.CertificateKey == "" {
			return fmt.Errorf("cfg.SSL.Cert.CertificateKey is required")
		}
	}

	httpsPort := c.cfg.HTTPSPort

	// http.ListenAndServeTLS(fmt.Sprintf(""::"%d", port), "", "", app)
	// srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: app}
	var minVersionTLS uint16 = 0x0301 // TLS 1.0
	server := &http.Server{
		ReadTimeout:  50 * time.Second,
		WriteTimeout: 600 * time.Second,
		IdleTimeout:  60 * time.Second,
		Addr:         fmt.Sprintf(":%d", httpsPort),
		TLSConfig: &tls.Config{
			MinVersion: minVersionTLS,
			GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
				if chi.ServerName == "" {
					return nil, fmt.Errorf("no server name (sni)")
				}

				// ssl
				if c.cfg.SSL != nil {
					var certificate string
					var certificateKey string

					// parts := strings.Split(chi.ServerName, ".")

					serverName := chi.ServerName
					for _, ssl := range c.cfg.SSL {
						// // extract match domain
						// // x.y.com.cn => x.y.com
						// // x.y.z.com => x.y.z.com
						// if ssl.Domain == chi.ServerName {
						// 	certificate = ssl.Cert.Certificate
						// 	certificateKey = ssl.Cert.CertificateKey
						// } else {
						// 	// x.y.com => y.com
						// 	// x.y.z.com => y.z.com => z.com
						// 	for i := 1; i <= len(parts)-2; i++ {
						// 		if ssl.Domain == strings.Join(parts[i:], ".") {
						// 			certificate = ssl.Cert.Certificate
						// 			certificateKey = ssl.Cert.CertificateKey
						// 			break
						// 		}
						// 	}
						// }

						if strings.EndsWith(serverName, ssl.Domain) {
							certificate = ssl.Cert.Certificate
							certificateKey = ssl.Cert.CertificateKey
							break
						}
					}

					if certificate == "" || certificateKey == "" {
						return nil, fmt.Errorf("no certificate")
					}

					cert, err := tls.LoadX509KeyPair(certificate, certificateKey)
					if err != nil {
						return nil, err
					}

					return &cert, nil
				}

				return nil, nil
			},
		},
		Handler: c.app,
	}

	c.app.Logger.Info("ingress start at https://127.0.0.1:%d", httpsPort)
	return server.ListenAndServeTLS("", "")
}
