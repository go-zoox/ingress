package core

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-zoox/ingress/core/service"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"github.com/yookoala/gofast"
)

// handleFastCGI handles a single HTTP request by forwarding it to php-fpm
// using the FastCGI protocol when service protocol is "fastcgi".
func (c *core) handleFastCGI(ctx *zoox.Context, serviceIns *service.Service) error {
	// php-fpm usually listens on tcp host:port or unix socket.
	// We reuse existing service configuration:
	//   Name: php-fpm address or unix socket path
	//   Port: php-fpm port (tcp). If Port is 0, treat Name as unix socket path.
	backend := serviceIns.Name

	var network, address string
	if serviceIns.Port == 0 {
		// unix socket
		network = "unix"
		address = backend
	} else {
		network = "tcp"
		address = net.JoinHostPort(backend, fmt.Sprintf("%d", serviceIns.Port))
	}

	connFactory := gofast.SimpleConnFactory(network, address)

	// Front controller:
	// - All dynamic requests最终落到单一入口脚本（默认 /index.php）
	// - 具体路由由应用自身根据 REQUEST_URI/QUERY_STRING 决定（符合 ThinkPHP 等框架习惯）
	//
	// scriptPath 是 php-fpm 侧看到的脚本路径，一般形如 "/index.php"。
	scriptPath := resolveFastCGIScriptPath(serviceIns)

	endpoint := gofast.NewFileEndpoint(scriptPath)(gofast.BasicSession)
	session := func(client gofast.Client, req *gofast.Request) (*gofast.ResponsePipe, error) {
		applyFrameworkFastCGIParams(req, serviceIns, scriptPath)
		return endpoint(client, req)
	}
	handler := gofast.NewHandler(session, gofast.SimpleClientFactory(connFactory))

	rec := &responseRecorder{ResponseWriter: ctx.Writer, statusCode: http.StatusOK}
	handler.ServeHTTP(rec, ctx.Request)

	if rec.statusCode >= 500 {
		logger.Errorf("fastcgi error: status=%d path=%s backend=%s", rec.statusCode, ctx.Path, address)
	}

	// Ensure X-Powered-By is set consistently
	ctx.Writer.Header().Del("X-Powered-By")
	ctx.Writer.Header().Set("X-Powered-By", fmt.Sprintf("gozoox-ingress/%s", c.version))

	// When using gofast handler, errors are already written to ResponseWriter.
	// We just return nil here so outer middleware treats it as handled.
	return nil
}

func resolveFastCGIScriptPath(serviceIns *service.Service) string {
	scriptPath := serviceIns.FastCGI.Script.Filename
	if scriptPath == "" {
		if strings.EqualFold(serviceIns.FastCGI.Framework, "thinkphp") {
			scriptPath = "/public/index.php"
		} else {
			scriptPath = "/index.php"
		}
	}
	if scriptPath[0] != '/' {
		scriptPath = "/" + scriptPath
	}

	return scriptPath
}

func applyFrameworkFastCGIParams(req *gofast.Request, serviceIns *service.Service, scriptPath string) {
	if !strings.EqualFold(serviceIns.FastCGI.Framework, "thinkphp") {
		return
	}

	rootDir := serviceIns.FastCGI.Script.RootDir
	if rootDir == "" {
		rootDir = "/var/www/html"
	}

	pathInfo := req.Raw.URL.Path
	if pathInfo == "" {
		pathInfo = "/"
	}
	if strings.HasPrefix(pathInfo, scriptPath) {
		pathInfo = strings.TrimPrefix(pathInfo, scriptPath)
		if pathInfo == "" {
			pathInfo = "/"
		}
	}

	req.Params["DOCUMENT_ROOT"] = rootDir
	req.Params["SCRIPT_NAME"] = scriptPath
	req.Params["SCRIPT_FILENAME"] = filepath.ToSlash(filepath.Join(rootDir, strings.TrimPrefix(scriptPath, "/")))
	req.Params["PATH_INFO"] = pathInfo
	req.Params["PATH_TRANSLATED"] = filepath.ToSlash(filepath.Join(rootDir, strings.TrimPrefix(pathInfo, "/")))
	req.Params["REQUEST_URI"] = req.Raw.URL.RequestURI()
}

// responseRecorder wraps http.ResponseWriter to capture status code.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
