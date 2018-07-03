package sdk

import (
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"os"
)

const activatePath = "/Plugin.Activate"

type muxInitilizaer func(*Handler)

type activationResposne struct {
	Implements []string
}

// Handler is the base to create plugin handlers.
// It initializes connections and sockets to listen to.
type Handler struct {
	mux     *http.ServeMux
	drivers map[string]muxInitilizaer
}

// NewHandler creates a new Handler with an http mux.
func NewHandler() *Handler {
	return &Handler{mux: http.NewServeMux(), drivers: make(map[string]muxInitilizaer)}
}

// RegisterDriver registers a Docker driver to the Handler
func (h *Handler) RegisterDriver(driver string, init muxInitilizaer) {
	h.drivers[driver] = init
}

// Serve sets up the handler to serve requests on the passed in listener
func (h *Handler) Serve(l net.Listener) error {
	activateResp := activationResposne{}
	for d, i := range h.drivers {
		activateResp.Implements = append(activateResp.Implements, d)
		i(h)
	}
	h.mux.HandleFunc(activatePath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", DefaultContentTypeV1_1)
		json.NewEncoder(w).Encode(activateResp)
	})

	server := http.Server{
		Addr:    l.Addr().String(),
		Handler: h.mux,
	}
	return server.Serve(l)
}

// ServeTCP makes the handler to listen for request in a given TCP address.
// It also writes the spec file in the right directory for docker to read.
// Due to constrains for running Docker in Docker on Windows, data-root directory
// of docker daemon must be provided. To get default directory, use
// WindowsDefaultDaemonRootDir() function. On Unix, this parameter is ignored.
func (h *Handler) ServeTCP(pluginName, addr, daemonDir string, tlsConfig *tls.Config) error {
	l, spec, err := newTCPListener(addr, pluginName, daemonDir, tlsConfig)
	if err != nil {
		return err
	}
	if spec != "" {
		defer os.Remove(spec)
	}
	return h.Serve(l)
}

// ServeUnix makes the handler to listen for requests in a unix socket.
// It also creates the socket file in the right directory for docker to read.
func (h *Handler) ServeUnix(addr string, gid int) error {
	l, spec, err := newUnixListener(addr, gid)
	if err != nil {
		return err
	}
	if spec != "" {
		defer os.Remove(spec)
	}
	return h.Serve(l)
}

// ServeWindows makes the handler to listen for request in a Windows named pipe.
// It also creates the spec file in the right directory for docker to read.
// Due to constrains for running Docker in Docker on Windows, data-root directory
// of docker daemon must be provided. To get default directory, use
// WindowsDefaultDaemonRootDir() function. On Unix, this parameter is ignored.
func (h *Handler) ServeWindows(addr, pluginName, daemonDir string, pipeConfig *WindowsPipeConfig) error {
	l, spec, err := newWindowsListener(addr, pluginName, daemonDir, pipeConfig)
	if err != nil {
		return err
	}
	if spec != "" {
		defer os.Remove(spec)
	}
	return h.Serve(l)
}

// HandleFunc registers a function to handle a request path with.
func (h *Handler) HandleFunc(path string, fn func(w http.ResponseWriter, r *http.Request)) {
	h.mux.HandleFunc(path, fn)
}
