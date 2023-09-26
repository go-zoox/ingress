package core

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (c *core) Run() error {
	if err := c.build(); err != nil {
		return err
	}

	https := c.cfg.HTTPSPort != 0
	httpsPort := c.cfg.HTTPSPort

	port := c.cfg.Port
	if port == 0 {
		port = 8080
	}

	c.app.Logger.Info("ingress start at http://127.0.0.1:%d", port)

	if https {
		c.app.Logger.Info("ingress start at https://127.0.0.1:%d", httpsPort)

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
						return nil, fmt.Errorf("no server name")
					}

					// ssl
					if c.cfg.SSL != nil {
						var certificate string
						var certificateKey string

						parts := strings.Split(chi.ServerName, ".")

						for _, ssl := range c.cfg.SSL {
							// extract match domain
							// x.y.com.cn => x.y.com
							// x.y.z.com => x.y.z.com
							if ssl.Domain == chi.ServerName {
								certificate = ssl.Cert.Certificate
								certificateKey = ssl.Cert.CertificateKey
							} else {
								// x.y.com => y.com
								// x.y.z.com => y.z.com => z.com
								for i := 1; i <= len(parts)-2; i++ {
									if ssl.Domain == strings.Join(parts[i:], ".") {
										certificate = ssl.Cert.Certificate
										certificateKey = ssl.Cert.CertificateKey
										break
									}
								}
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

		go func() {
			if err := server.ListenAndServeTLS("", ""); err != nil {
				panic(err)
			}
		}()
	}

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), c.app); err != nil {
		return err
	}

	return nil
}
