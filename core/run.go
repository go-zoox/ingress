package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-zoox/chalk"
	"github.com/go-zoox/core-utils/strings"
	"github.com/go-zoox/ingress"
)

func (c *core) Run() error {
	if err := c.build(); err != nil {
		return err
	}

	// g, ctx := errgroup.WithContext(context.Background())

	// g.Go(func() error {
	// 	return c.serveHTTP(ctx)
	// })

	// g.Go(func() error {
	// 	return c.serveHTTPs(ctx)
	// })

	// return g.Wait()

	c.app.SetBanner(fmt.Sprintf(`
   ____                        
  /  _/__  ___ ________ ___ ___
 _/ // _ \/ _ '/ __/ -_|_-<(_-<
/___/_//_/\_, /_/  \__/___/___/
         /___/                 
				 			 
%s %s

____________________________________O/_______
                                    O\
`, chalk.Green("Ingress"), chalk.Green("v"+ingress.Version)))

	c.app.SetTLSCertLoader(func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if chi.ServerName == "" {
			return nil, fmt.Errorf("no server name (sni)")
		}

		// ssl
		if c.cfg.SSL != nil {
			var certificate string
			var certificateKey string

			serverName := chi.ServerName
			for _, ssl := range c.cfg.SSL {
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
	})

	return c.app.Run()
}

func (c *core) serveHTTP(ctx context.Context) error {
	port := c.cfg.Port
	if port == 0 {
		port = 8080
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: c.app,
	}

	go func() {
		<-ctx.Done() // 当上下文被取消时，停止服务器
		server.Close()
	}()

	c.app.Logger.Info("ingress start at http://127.0.0.1:%d", port)
	return server.Serve(listener)
}

func (c *core) serveHTTPs(ctx context.Context) error {
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

	go func() {
		<-ctx.Done() // 当上下文被取消时，停止服务器
		server.Close()
	}()

	c.app.Logger.Info("ingress start at https://127.0.0.1:%d", httpsPort)
	return server.ListenAndServeTLS("", "")
}
