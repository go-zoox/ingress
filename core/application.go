package core

// References:
//   https://www.cnblogs.com/zyndev/p/14454891.html
//   https://h1z3y3.me/posts/simple-and-powerful-reverse-proxy-in-golang/
//   https://segmentfault.com/a/1190000039778241

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-zoox/logger"
	"github.com/go-zoox/proxy"
	"github.com/go-zoox/uuid"
)

type Application struct {
	Version string
	Config  *Config

	//
	middlewares []Middleware

	Logger *logger.Logger
}

func New(version string, cfg *Config) *Application {
	return &Application{
		Version: version,
		Logger:  logger.New(),
		Config:  cfg,
	}
}

// func (app *Application) Load() error {
// 	var cfg Config
// 	if err := config.Load(&cfg, &config.LoadOptions{
// 		FilePath: "conf/ingress.yaml",
// 	}); err != nil {
// 		return err
// 	}

// 	app.Config = &cfg
// 	return nil
// }

func (app *Application) Use(middleware ...Middleware) {
	app.middlewares = append(app.middlewares, middleware...)
}

func (app *Application) createContext() *Context {
	onRequest := func(ctx *Context) error {
		for _, middleware := range app.middlewares {
			if err := middleware.OnRequest(ctx); err != nil {
				return err
			}
		}

		return nil
	}

	onResponse := func(ctx *Context) error {
		for _, middleware := range app.middlewares {
			if err := middleware.OnResponse(ctx); err != nil {
				return err
			}
		}

		return nil
	}

	return &Context{
		// Application: app,
		onRequest:  onRequest,
		onResponse: onResponse,
		Logger:     app.Logger,
	}
}

func (app *Application) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// health check
	if req.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := app.createContext()

	p := proxy.New(&proxy.Config{
		OnRequest: func(req *http.Request) error {
			ctx.Request = NewRequest(req)

			host, _ := proxy.ParseHostPort(ctx.Request.Host)
			ctx.Host = host
			ctx.Path = req.URL.Path
			ctx.Method = req.Method
			ctx.RequestId = uuid.V4()
			ctx.RequestStartAt = time.Now()

			// logger.Info("[request] %s %s", req.Method, req.URL.String())

			ctx.onRequest(ctx)

			return nil
		},
		OnResponse: func(res *http.Response) error {
			ctx.Response = NewResponse(res)

			ctx.Status = res.StatusCode
			ctx.RequestEndAt = time.Now()
			// ctx.Logger.Info("[response] %d %s", res.StatusCode, res.Status)

			ctx.onResponse(ctx)
			return nil
		},
	})

	p.ServeHTTP(w, req)
}

func (app *Application) Start() {
	https := app.Config.HTTPSPort != 0
	httpsPort := app.Config.HTTPSPort

	port := app.Config.Port
	if port == 0 {
		port = 8080
	}

	app.Logger.Info("ingress start at http://127.0.0.1:%d", port)

	if https {
		app.Logger.Info("ingress start at https://127.0.0.1:%d", httpsPort)

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
					if app.Config.SSL != nil {
						var certificate string
						var certificateKey string

						parts := strings.Split(chi.ServerName, ".")

						for _, ssl := range app.Config.SSL {
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
			Handler: app,
		}

		go server.ListenAndServeTLS("", "")
	}

	http.ListenAndServe(fmt.Sprintf(":%d", port), app)
}
