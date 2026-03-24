package core

const (
	// backend type selector
	backendTypeService = "service"
	backendTypeHandler = "handler"

	// handler type selector when backend.type=handler
	handlerTypeStaticResponse = "static_response"
	handlerTypeFileServer     = "file_server"
	handlerTypeTemplates      = "templates"
)
