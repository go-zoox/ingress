package core

import "net/http"

type Response struct {
	*http.Response
}

func NewResponse(res *http.Response) *Response {
	return &Response{res}
}
