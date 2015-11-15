package sleepy

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

const (
	// GET HTTP method
	GET = "GET"
	// POST HTTP method
	POST = "POST"
	// PUT HTTP method
	PUT = "PUT"
	// DELETE HTTP method
	DELETE = "DELETE"
	// HEAD HTTP method
	HEAD = "HEAD"
	// PATCH HTTP method
	PATCH = "PATCH"
)

var (
	httpReadTimeout  uint
	httpWriteTimeout uint
)

func init() {
	flag.UintVar(&httpReadTimeout, "httpReadTimeout", 20, "sleepy: Specifies the ReadTimeout.")
	flag.UintVar(&httpWriteTimeout, "httpWriteTimeout", 20, "sleepy: Specifies the WriteTimeout.")
}

// GetSupported is the interface that provides the Get
// method a resource must support to receive HTTP GETs.
type GetSupported interface {
	Get(*http.Request, http.Header, httprouter.Params) (int, interface{}, http.Header)
}

// PostSupported is the interface that provides the Post
// method a resource must support to receive HTTP POSTs.
type PostSupported interface {
	Post(*http.Request, http.Header, httprouter.Params) (int, interface{}, http.Header)
}

// PutSupported is the interface that provides the Put
// method a resource must support to receive HTTP PUTs.
type PutSupported interface {
	Put(*http.Request, http.Header, httprouter.Params) (int, interface{}, http.Header)
}

// DeleteSupported is the interface that provides the Delete
// method a resource must support to receive HTTP DELETEs.
type DeleteSupported interface {
	Delete(*http.Request, http.Header, httprouter.Params) (int, interface{}, http.Header)
}

// HeadSupported is the interface that provides the Head
// method a resource must support to receive HTTP HEADs.
type HeadSupported interface {
	Head(*http.Request, http.Header, httprouter.Params) (int, interface{}, http.Header)
}

// PatchSupported is the interface that provides the Patch
// method a resource must support to receive HTTP PATCHs.
type PatchSupported interface {
	Patch(*http.Request, http.Header, httprouter.Params) (int, interface{}, http.Header)
}

// API is the interface to manage a group of resources by routing requests
// to the correct method on a matching resource and marshalling
// the returned data to JSON for the HTTP response.
type API interface {
	// Mux returns the Mux used by an API. If a Mux has
	// does not yet exist, a new one will be created and returned.
	Mux() *httprouter.Router
	// AddResource adds a new resource to an API. The API will route
	// requests that match one of the given paths to the matching HTTP
	// method on the resource.
	AddResource(resource interface{}, paths ...string)
	// AddResourceWithWrapper behaves exactly like AddResource but wraps
	// the generated handler function with a give wrapper function to allow
	// to hook in Gzip support and similar.
	AddResourceWithWrapper(resource interface{}, wrapper func(handler httprouter.Handle) httprouter.Handle, paths ...string)
	// Start causes the API to begin serving requests on the given port.
	Start(port int) error
	// SetMux sets the Mux to use by an API.
	SetMux(mux *httprouter.Router) error
	// SetLogger sets log.Logger for loging all requests
	SetLogger(logger *log.Logger)
}

// An DefaultAPI manages a group of resources by routing requests
// to the correct method on a matching resource and marshalling
// the returned data to JSON for the HTTP response.
//
// You can instantiate multiple APIs on separate ports. Each API
// will manage its own set of resources.
type DefaultAPI struct {
	Logger *log.Logger

	mux            *httprouter.Router
	muxInitialized bool
}

// NewAPI allocates and returns a new API.
func NewAPI(options ...func(*DefaultAPI)) API {

	api := DefaultAPI{}

	// set any options
	for _, o := range options {
		o(&api)
	}

	return &api
}

// SetLogger sets log.Logger for loging all requests
func (api *DefaultAPI) SetLogger(logger *log.Logger) {
	api.Logger = logger
}

func (api *DefaultAPI) requestHandler(resource interface{}) httprouter.Handle {
	return func(rw http.ResponseWriter, request *http.Request, params httprouter.Params) {

		if request.ParseForm() != nil {
			api.logRequest(request, http.StatusBadRequest, "request.ParseForm was nil")
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		var handler func(*http.Request, http.Header, httprouter.Params) (int, interface{}, http.Header)

		switch request.Method {
		case GET:
			if resource, ok := resource.(GetSupported); ok {
				handler = resource.Get
			}
		case POST:
			if resource, ok := resource.(PostSupported); ok {
				handler = resource.Post
			}
		case PUT:
			if resource, ok := resource.(PutSupported); ok {
				handler = resource.Put
			}
		case DELETE:
			if resource, ok := resource.(DeleteSupported); ok {
				handler = resource.Delete
			}
		case HEAD:
			if resource, ok := resource.(HeadSupported); ok {
				handler = resource.Head
			}
		case PATCH:
			if resource, ok := resource.(PatchSupported); ok {
				handler = resource.Patch
			}
		}

		if handler == nil {
			api.logRequest(request, http.StatusMethodNotAllowed, "Handler was nil")
			rw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		code, data, header := handler(request, request.Header, params)
		api.logRequest(request, code, "OK")

		content, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			api.logRequest(request, http.StatusInternalServerError, "err in json.MarshalIndent: %s", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		for name, values := range header {
			for _, value := range values {
				rw.Header().Add(name, value)
			}
		}
		rw.WriteHeader(code)
		rw.Write(content)
	}
}

// Mux returns the Mux used by an API. If a Mux has
// does not yet exist, a new one will be created and returned.
func (api *DefaultAPI) Mux() *httprouter.Router {
	if api.muxInitialized {
		return api.mux
	}

	api.mux = httprouter.New()
	api.muxInitialized = true

	// TODO log 404
	//

	return api.mux
}

// SetMux sets the Mux to use by an API.
func (api *DefaultAPI) SetMux(mux *httprouter.Router) error {
	if api.muxInitialized {
		return errors.New("You cannot set a muxer when already initialized.")
	}
	api.mux = mux
	api.muxInitialized = true
	return nil
}

// AddResource adds a new resource to an API. The API will route
// requests that match one of the given paths to the matching HTTP
// method on the resource.
func (api *DefaultAPI) AddResource(resource interface{}, paths ...string) {
	for _, path := range paths {
		if resource, ok := resource.(GetSupported); ok {
			api.Mux().GET(path, api.requestHandler(resource))
		}
		if resource, ok := resource.(PostSupported); ok {
			api.Mux().POST(path, api.requestHandler(resource))
		}
		if resource, ok := resource.(PutSupported); ok {
			api.Mux().PUT(path, api.requestHandler(resource))
		}
		if resource, ok := resource.(DeleteSupported); ok {
			api.Mux().DELETE(path, api.requestHandler(resource))
		}
		if resource, ok := resource.(HeadSupported); ok {
			api.Mux().HEAD(path, api.requestHandler(resource))
		}
		if resource, ok := resource.(PatchSupported); ok {
			api.Mux().PATCH(path, api.requestHandler(resource))
		}
	}
}

// AddResourceWithWrapper behaves exactly like AddResource but wraps
// the generated handler function with a give wrapper function to allow
// to hook in Gzip support and similar.
func (api *DefaultAPI) AddResourceWithWrapper(resource interface{}, wrapper func(handler httprouter.Handle) httprouter.Handle, paths ...string) {
	for _, path := range paths {
		if resource, ok := resource.(GetSupported); ok {
			api.Mux().GET(path, wrapper(api.requestHandler(resource)))
		}
		if resource, ok := resource.(PostSupported); ok {
			api.Mux().POST(path, wrapper(api.requestHandler(resource)))
		}
		if resource, ok := resource.(PutSupported); ok {
			api.Mux().PUT(path, wrapper(api.requestHandler(resource)))
		}
		if resource, ok := resource.(DeleteSupported); ok {
			api.Mux().DELETE(path, wrapper(api.requestHandler(resource)))
		}
		if resource, ok := resource.(HeadSupported); ok {
			api.Mux().HEAD(path, wrapper(api.requestHandler(resource)))
		}
		if resource, ok := resource.(PatchSupported); ok {
			api.Mux().PATCH(path, wrapper(api.requestHandler(resource)))
		}

	}
}

// Start causes the API to begin serving requests on the given port.
func (api *DefaultAPI) Start(port int) error {
	if !api.muxInitialized {
		err := errors.New("You must add at least one resource to this API.")
		api.log(err.Error())
		return err
	}
	portString := fmt.Sprintf(":%d", port)
	api.log("Listening on http://localhost:%d", port)

	server := &http.Server{
		Addr:           portString,
		Handler:        api.Mux(),
		ReadTimeout:    20 * time.Second,
		WriteTimeout:   20 * time.Second,
		MaxHeaderBytes: 1 << 15,
	}

	return server.ListenAndServe()
}

func (api *DefaultAPI) log(msg string, args ...interface{}) {

	if api.Logger == nil {
		return
	}

	m := msg
	if len(args) > 0 {
		m = fmt.Sprintf(m, args...)
	}
	api.Logger.Println("sleepy:", m)
}

func (api *DefaultAPI) logRequest(r *http.Request, code int, msg string, args ...interface{}) {

	m := msg
	if len(args) > 0 {
		m = fmt.Sprintf(msg, args...)
	}

	api.log("[%v] %s %s%s %d, %s", r.RemoteAddr, r.Method, r.URL.Path, r.URL.RawQuery, code, m)
}
