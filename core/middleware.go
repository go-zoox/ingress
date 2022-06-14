package core

// type Middleware interface {
// 	OnRequest(ctx *Context) error
// 	OnResponse(ctx *Context) error
// }

type Middleware interface {
	OnRequest(ctx *Context) error
	OnResponse(ctx *Context) error
}
