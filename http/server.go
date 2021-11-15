package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Ctx struct {
	*http.Request
	http.ResponseWriter
}

func (c *Ctx) Vars(name string) string {
	v := mux.Vars(c.Request)
	if v != nil {
		return v[name]
	}
	return ""
}
func (c *Ctx) Header(name string) string {
	return c.Request.Header.Get(name)
}
func (c *Ctx) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}
func (c *Ctx) Method() string {
	return c.Request.Method
}
func (c *Ctx) Body() (*gabs.Container, error) {
	b := c.Request.Body
	defer b.Close()
	buf, err := ioutil.ReadAll(b)
	if err != nil {
		return nil, err
	}
	return gabs.ParseJSON(buf)
}

func (c *Ctx) SetHeader(name, value string) {
	c.ResponseWriter.Header().Set(name, value)
}
func (c *Ctx) SetStatus(code int) {
	c.ResponseWriter.WriteHeader(code)
}
func (c *Ctx) SendJson(data *gabs.Container) {
	c.SetHeader("Content-Type", "application/json")
	_, _ = c.ResponseWriter.Write(data.Bytes())
}
func (c *Ctx) SendString(data string) {
	_, _ = c.ResponseWriter.Write([]byte(data))
}
func (c *Ctx) SendFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, _ = io.Copy(c.ResponseWriter, file)
	return nil
}

type Server struct {
	status uint32
	mux    sync.Mutex
	Addr   string
	*http.Server
	*mux.Router
	log func(string)
}

func NewServer(addr string, log func(string)) *Server {
	return &Server{Addr: addr, Router: mux.NewRouter(), log: log}
}

func (s *Server) Stop(n time.Duration) (error, context.CancelFunc) {
	if s.Server == nil {
		return errors.New("not started"), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), n)
	go func() {
		err := s.Shutdown(ctx)
		if err != nil {

		}
	}()
	return nil, cancel
}
func (s *Server) Running() bool {
	return s.status != 0
}
func (s *Server) Start(
	cors bool,
	allowHeader []string,
	allowedMethods []string,
	allowedOrigins []string,
	exposedHeaders []string,
	maxAge int,
	allowCredentials int,
) {
	if !cors {
		s.Server = &http.Server{
			Addr:    s.Addr,
			Handler: handlers.RecoveryHandler()(s.Router),
		}
	} else if allowHeader == nil &&
		allowedMethods == nil &&
		allowedOrigins == nil &&
		exposedHeaders == nil &&
		maxAge == 0 &&
		allowCredentials == 0 {
		s.Server = &http.Server{
			Addr:    s.Addr,
			Handler: handlers.CORS()(handlers.RecoveryHandler()(s.Router)),
		}
	} else {
		opt := make([]handlers.CORSOption, 0)
		if allowHeader != nil {
			opt = append(opt, handlers.AllowedHeaders(allowHeader))
		}
		if allowedMethods != nil {
			opt = append(opt, handlers.AllowedMethods(allowedMethods))
		}
		if allowedOrigins != nil {
			opt = append(opt, handlers.AllowedOrigins(allowedOrigins))
		}
		if exposedHeaders != nil {
			opt = append(opt, handlers.ExposedHeaders(exposedHeaders))
		}
		if maxAge > 0 {
			opt = append(opt, handlers.MaxAge(maxAge))
		}
		if allowCredentials > 0 {
			opt = append(opt, handlers.AllowCredentials())
		}
		s.Server = &http.Server{
			Addr:    s.Addr,
			Handler: handlers.CORS(opt...)(handlers.RecoveryHandler()(s.Router)),
		}
	}

	s.Server.RegisterOnShutdown(func() {
		if v := atomic.LoadUint32(&s.status); v != 0 {
			s.mux.Lock()
			defer atomic.StoreUint32(&s.status, 0)
			defer s.mux.Unlock()
			if s.status != 0 {
				s.Server = nil
			}
		}
	})
	go func() {
		if v := atomic.LoadUint32(&s.status); v != 1 {
			s.mux.Lock()
			if s.status != 1 {
				atomic.StoreUint32(&s.status, 1)
			}
			s.mux.Unlock()
		}
		log.Println("http start at ", s.Addr)
		if err := s.ListenAndServe(); err != nil && !errors.Is(http.ErrServerClosed, err) && s.log != nil {
			s.log(fmt.Sprintf("close server fail: %s", err))
		}
	}()
}

func (s *Server) Route(path string, fn func(ctx *Ctx)) {
	s.Router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		fn(&Ctx{r, w})
	})
}
func (s *Server) Get(path string, fn func(ctx *Ctx)) {
	s.Router.
		HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			fn(&Ctx{r, w})
		}).
		Methods(http.MethodGet)
}
func (s *Server) Post(path string, fn func(ctx *Ctx)) {
	s.Router.
		HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			fn(&Ctx{r, w})
		}).
		Methods(http.MethodPost)
}
func (s *Server) Put(path string, fn func(ctx *Ctx)) {
	s.Router.
		HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			fn(&Ctx{r, w})
		}).
		Methods(http.MethodPut)
}
func (s *Server) Head(path string, fn func(ctx *Ctx)) {
	s.Router.
		HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			fn(&Ctx{r, w})
		}).
		Methods(http.MethodHead)
}
func (s *Server) Patch(path string, fn func(ctx *Ctx)) {
	s.Router.
		HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			fn(&Ctx{r, w})
		}).
		Methods(http.MethodPatch)
}
func (s *Server) Delete(path string, fn func(ctx *Ctx)) {
	s.Router.
		HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			fn(&Ctx{r, w})
		}).
		Methods(http.MethodDelete)
}
func (s *Server) Connect(path string, fn func(ctx *Ctx)) {
	s.Router.
		HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			fn(&Ctx{r, w})
		}).
		Methods(http.MethodConnect)
}
func (s *Server) Options(path string, fn func(ctx *Ctx)) {
	s.Router.
		HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			fn(&Ctx{r, w})
		}).
		Methods(http.MethodOptions)
}
func (s *Server) Trace(path string, fn func(ctx *Ctx)) {
	s.Router.
		HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			fn(&Ctx{r, w})
		}).
		Methods(http.MethodTrace)
}
func (s *Server) File(path, prefix, dir string) {
	s.Router.Handle(path, http.StripPrefix(prefix, http.FileServer(http.Dir(dir))))
}
