package core

import (
	"time"

	"github.com/go-zoox/logger"
)

type Context struct {
	// Application *application.Application
	Request  *Request
	Response *Response
	//
	onRequest  func(ctx *Context) error
	onResponse func(ctx *Context) error
	//
	Logger *logger.Logger
	//
	RequestStartAt time.Time
	RequestEndAt   time.Time
	RequestId      string
	//
	Host   string
	Method string
	Path   string
	//
	Status int
}

func (c *Context) RequestTime() time.Duration {
	return c.RequestEndAt.Sub(c.RequestStartAt)
}
