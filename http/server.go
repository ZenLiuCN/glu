package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	. "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
	"glu"
	gabs2 "glu/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	CtxType    *glu.Type
	ServerType *glu.Type
	HttpModule *glu.Module
)

func init() {
	HttpModule = glu.NewModular("http", `http module built on net/http gorilla/mux, requires json module.
http.Ctx    ctx type is an wrap on http.Request and http.ResponseWriter, should never call new!
http.Server server type is wrap with mux.Router and http.Server.
IMPORTANT: Server handler should be an independent lua function in string form, minimal sample as below:
local http=require('http')
local json=require('json')
local server=http.Server.new(':8081')
server:get('/',[[
local c=...
c:sendJson(c:query('p'))
]])
server:start(false)
while (true) do	end
`, true)
	chkCtx := func(s *LState) *Ctx {
		ud := s.CheckUserData(1)
		if v, ok := ud.Value.(*Ctx); ok {
			return v
		}
		s.ArgError(1, "http.Ctx expected")
		return nil
	}
	CtxType = glu.NewType("Ctx", false, ``,
		func(s *LState) interface{} {
			s.RaiseError("not allow to create ctx instance")
			return nil
		}).
		AddMethod("vars", `Ctx:vars(name string)string ==> fetch path variable`,
			func(s *LState) int {
				v := chkCtx(s)
				s.CheckType(2, LTString)
				s.Push(LString(v.Vars(s.ToString(2))))
				return 1
			}).
		AddMethod("header", `Ctx:header(name string)string ==> fetch request header`,
			func(s *LState) int {
				v := chkCtx(s)
				s.CheckType(2, LTString)
				s.Push(LString(v.Header(s.ToString(2))))
				return 1
			}).
		AddMethod("query", `Ctx:query(name string)string ==> fetch request query parameter`,
			func(s *LState) int {
				v := chkCtx(s)
				s.Push(LString(v.Query(s.CheckString(2))))
				return 1
			}).
		AddMethod("method", `Ctx:method()string ==> fetch request method`,
			func(s *LState) int {
				v := chkCtx(s)
				s.Push(LString(v.Method()))
				return 1
			}).
		AddMethod("body", `Ctx:body()json.Json ==> fetch request body`,
			func(s *LState) int {
				v := chkCtx(s)
				body, err := v.Body()
				if err != nil {
					s.RaiseError("fetch body error %s", err)
					return 0
				}
				return gabs2.JsonType.New(s, body)
			}).
		AddMethod("setHeader", `Ctx:setHeader(key,value string)Ctx ==> chain method set output header`,
			func(s *LState) int {
				ud := s.ToUserData(1)
				v := chkCtx(s)
				s.CheckTypes(2, LTString)
				s.CheckTypes(3, LTString)
				v.SetHeader(s.ToString(2), s.ToString(3))
				s.Push(ud)
				return 1
			}).
		AddMethod("sendStatus", `Ctx:sendStatus(code int) ==> send http status,this will close process`,
			func(s *LState) int {
				v := chkCtx(s)
				s.CheckTypes(2, LTNumber)
				v.SendStatus(s.ToInt(2))
				return 0
			}).
		AddMethod("send", `Ctx:send(json json.Json) ==> send json body,this will close process`,
			func(s *LState) int {
				v := chkCtx(s)
				g := gabs2.JsonTypeCheck(s, 2)
				v.Send(g)
				return 0
			}).
		AddMethod("sendFile", `Ctx:sendFile(path string) ==> send file,this will close process`,
			func(s *LState) int {
				v := chkCtx(s)
				s.CheckTypes(2, LTString)
				err := v.SendFile(s.ToString(2))
				if err != nil {
					s.RaiseError("send file %s", err)
					return 0
				}
				return 0
			})
	chkServer := func(s *LState) *Server {
		ud := s.CheckUserData(1)
		if v, ok := ud.Value.(*Server); ok {
			return v
		}
		s.ArgError(1, "http.Server expected")
		return nil
	}
	ServerType = glu.NewType("Server", false, `Server.new(addr string)`,
		func(s *LState) interface{} {
			s.CheckType(1, LTString)
			return NewServer(s.ToString(1), func(s string) {
				//TODO
				log.Println(s)
			})
		}).
		AddMethod("stop", `Server:stop(seconds int) ==> stop server graceful`,
			func(s *LState) int {
				v := chkServer(s)
				s.CheckType(2, LTNumber)
				err, _ := v.Stop(time.Second * time.Duration(s.ToInt(2)))
				if err != nil {
					s.RaiseError("stop http.Server fail %s", err)
				}
				return 0
			}).
		AddMethod("running", `Server:running()bool ==> check server is running`,
			func(s *LState) int {
				v := chkServer(s)
				if v.Running() {
					s.Push(LTrue)
				} else {
					s.Push(LFalse)
				}
				return 1
			}).
		AddMethod("start", `Server:start(
    cors bool,                          ==> enable cors or not,default false.
	allowHeader    []string?,           ==> cors config for header allowed.
	allowedMethods []string?,           ==> cors config for methods allowed.
	allowedOrigins []string?,           ==> cors config for origin allowed.
	exposedHeaders []string?,           ==> cors config for header exposed.
	maxAge int?,                        ==> cors config for maxAge ,maximum 600 seconds.
	allowCredentials bool?,             ==> cors config for allowCredentials.
)                                       ==> start server,should only call once.`,
			func(s *LState) int {
				v := chkServer(s)
				c := s.CheckBool(2)
				if !c {
					v.Start(false, nil, nil, nil, nil, 0, 0)
				} else if s.GetTop() == 2 {
					v.Start(true, tableToSlice(s, 2), nil, nil, nil, 0, 0)
				} else if s.GetTop() == 3 {
					v.Start(true, tableToSlice(s, 2), tableToSlice(s, 3), nil, nil, 0, 0)
				} else if s.GetTop() == 4 {
					v.Start(true, tableToSlice(s, 2), tableToSlice(s, 3),
						tableToSlice(s, 4), nil, 0, 0)
				} else if s.GetTop() == 5 {
					v.Start(true, tableToSlice(s, 2), tableToSlice(s, 3),
						tableToSlice(s, 4), tableToSlice(s, 5), 0, 0)
				} else if s.GetTop() == 6 {
					v.Start(true, tableToSlice(s, 2), tableToSlice(s, 3),
						tableToSlice(s, 4), tableToSlice(s, 5), s.CheckInt(6), 0)
				} else if s.GetTop() == 7 {
					b := s.CheckBool(7)
					t := -1
					if b {
						t = 1
					}
					v.Start(true, tableToSlice(s, 2), tableToSlice(s, 3),
						tableToSlice(s, 4), tableToSlice(s, 5), s.CheckInt(6), t)
				} else {
					s.RaiseError("invalid arguments for http.Server:start")
				}
				return 0
			}).
		AddMethod("route", `Server:route(path string,  ==> code should be string code slice with function handle http.Ctx
code string) ==> register handler without method limit.`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				code := s.CheckString(3)
				chunk, err := compile(code, route)
				if err != nil {
					s.RaiseError("compile handler fail %s", err)
					return 0
				}
				v.Route(route, func(ctx *Ctx) {
					x := glu.Get()
					defer glu.Put(x)
					fn := x.NewFunctionFromProto(chunk)
					x.Push(fn)
					_ = CtxType.New(x, ctx)
					err = x.PCall(1, 0, nil)
					if err != nil {
						ctx.SendStatus(500)
						return
					}
				})
				return 0
			}).
		AddMethod("get", `Server:get(path string, handler string) ==> register handler limit with GET.`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				code := s.CheckString(3)
				chunk, err := compile(code, route)
				if err != nil {
					s.RaiseError("compile handler fail %s", err)
					return 0
				}
				v.Get(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethod("post", `Server:post(path string, handler string) ==> register handler limit with POST.`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				code := s.CheckString(3)
				chunk, err := compile(code, route)
				if err != nil {
					s.RaiseError("compile handler fail %s", err)
					return 0
				}
				v.Post(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethod("put", `Server:put(path string, handler string) ==> register handler limit with POST.`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				code := s.CheckString(3)
				chunk, err := compile(code, route)
				if err != nil {
					s.RaiseError("compile handler fail %s", err)
					return 0
				}
				v.Put(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethod("head", `Server:head(path string,handler string) ==> register handler limit with HEAD.`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				code := s.CheckString(3)
				chunk, err := compile(code, route)
				if err != nil {
					s.RaiseError("compile handler fail %s", err)
					return 0
				}
				v.Head(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethod("patch", `Server:patch(path string,handler string) ==> register handler limit with PATCH.`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				code := s.CheckString(3)
				chunk, err := compile(code, route)
				if err != nil {
					s.RaiseError("compile handler fail %s", err)
					return 0
				}
				v.Patch(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethod("delete", `Server:delete(path string,handler string) ==> register handler limit with DELETE.`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				code := s.CheckString(3)
				chunk, err := compile(code, route)
				if err != nil {
					s.RaiseError("compile handler fail %s", err)
					return 0
				}
				v.Delete(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethod("connect", `Server:connect(path string,handler string) ==> register handler limit with CONNECT.`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				code := s.CheckString(3)
				chunk, err := compile(code, route)
				if err != nil {
					s.RaiseError("compile handler fail %s", err)
					return 0
				}
				v.Connect(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethod("options", `Server:options(path string,handler string) ==> register handler limit with OPTIONS.should not use if with cors enable`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				code := s.CheckString(3)
				chunk, err := compile(code, route)
				if err != nil {
					s.RaiseError("compile handler fail %s", err)
					return 0
				}
				v.Options(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethod("trace", `Server:trace(path string,handler string) ==> register handler limit with TRACE.`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				code := s.CheckString(3)
				chunk, err := compile(code, route)
				if err != nil {
					s.RaiseError("compile handler fail %s", err)
					return 0
				}
				v.Trace(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethod("files", `Server:files(path ,prefix, dir string) ==> register path to service with dir`,
			func(s *LState) int {
				v := chkServer(s)
				route := s.CheckString(2)
				pfx := s.CheckString(3)
				file := s.CheckString(4)
				v.File(route, pfx, file)
				return 0
			})

	HttpModule.AddModule(CtxType)
	HttpModule.AddModule(ServerType)
	glu.Registry = append(glu.Registry, HttpModule)
}
func executeHandler(chunk *FunctionProto, c *Ctx) {
	x := glu.Get()
	defer glu.Put(x)
	fn := x.NewFunctionFromProto(chunk)
	x.Push(fn)
	_ = CtxType.New(x, c)
	err := x.PCall(1, 0, nil)
	if err != nil {
		c.SendStatus(500)
		g := gabs.New()
		g.Set(err.Error(), "error")
		c.Send(g)
		fmt.Printf("error handle %+v : %s", c.URL, err)
		return
	}
}
func compile(code string, path string) (*FunctionProto, error) {
	name := fmt.Sprintf("handler(%s)", path)
	chunk, err := parse.Parse(strings.NewReader(code), name)
	if err != nil {
		return nil, err
	}
	return Compile(chunk, name)
}
func tableToSlice(s *LState, n int) (r []string) {
	v := s.Get(n)
	if v.Type() == LTNil {
		return
	}
	t := s.CheckTable(n)
	if t == nil {
		return
	}
	t.ForEach(func(k LValue, v LValue) {
		r = append(r, v.String())
	})
	return
}

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
func (c *Ctx) SendStatus(code int) {
	c.ResponseWriter.WriteHeader(code)
}
func (c *Ctx) Send(data *gabs.Container) {
	c.SetHeader("Content-Type", "application/json")
	_, _ = c.ResponseWriter.Write(data.Bytes())
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
