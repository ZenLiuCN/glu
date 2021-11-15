package http

import (
	"fmt"
	"github.com/Jeffail/gabs/v2"
	. "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
	"glu"
	"glu/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	CtxType    *glu.Type
	ServerType *glu.Type
	ResType    *glu.Type
	ClientType *glu.Type
	HttpModule *glu.Module
)

func init() {
	HttpModule = glu.NewModular("http", `http module built on net/http gorilla/mux, requires json module.
http.Ctx       ctx type is an wrap on http.Request and http.ResponseWriter, should never call new!
http.Res       res type is an wrap on http.Response , should never call new!
http.Client    client type is an wrap of http.Client.
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
	//region Ctx
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
				return json.JsonType.New(s, body)
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
		AddMethod("setStatus", `Ctx:setStatus(code int) ==> send http status,this will close process`,
			func(s *LState) int {
				v := chkCtx(s)
				s.CheckTypes(2, LTNumber)
				v.SetStatus(s.ToInt(2))
				return 0
			}).
		AddMethod("sendJson", `Ctx:sendJson(json json.Json) ==> send json body,this will close process`,
			func(s *LState) int {
				v := chkCtx(s)
				g := json.JsonTypeCheck(s, 2)
				v.SendJson(g)
				return 0
			}).
		AddMethod("sendString", `Ctx:sendString(text string) ==> send text body,this will close process`,
			func(s *LState) int {
				v := chkCtx(s)
				g := s.CheckString(2)
				v.SendString(g)
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
	//endregion
	//region Server
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
					executeHandler(chunk, ctx)
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
	//endregion
	chkRes := func(s *LState) *http.Response {
		ud := s.CheckUserData(1)
		if v, ok := ud.Value.(*http.Response); ok {
			return v
		}
		s.ArgError(1, "http.Res expected")
		return nil
	}
	ResType = glu.NewType("Res", false, ``,
		func(s *LState) interface{} {
			s.RaiseError("not allow to create ctx instance")
			return nil
		}).
		AddMethod("statusCode", `Res:statusCode()int`,
			func(s *LState) int {
				c := chkRes(s)
				s.Push(LNumber(c.StatusCode))
				return 1
			}).
		AddMethod("status", `Res:status()string`,
			func(s *LState) int {
				c := chkRes(s)
				s.Push(LString(c.Status))
				return 1
			}).
		AddMethod("contentLen", `Res:contentLen()int`,
			func(s *LState) int {
				c := chkRes(s)
				s.Push(LNumber(c.ContentLength))
				return 1
			}).
		AddMethod("header", `Res:header()map[string]string`,
			func(s *LState) int {
				c := chkRes(s)
				t := s.NewTable()
				for k, v := range c.Header {
					t.RawSetString(k, LString(strings.Join(v, ",")))
				}
				s.Push(t)
				return 1
			}).
		AddMethod("body", `Res:body()string ==> read body as string,body should only read once.`,
			func(s *LState) int {
				c := chkRes(s)
				defer c.Body.Close()
				buf, err := ioutil.ReadAll(c.Body)
				if err != nil {
					s.RaiseError("read body %s", err)
					return 0
				}
				s.Push(LString(buf))
				return 1
			}).
		AddMethod("bodyJson", `Res:bodyJson()(json.Json?,string?) ==> read body as Json,body should only read once.`,
			func(s *LState) int {
				c := chkRes(s)
				defer c.Body.Close()
				buf, err := ioutil.ReadAll(c.Body)
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
					return 2
				}
				g, err := gabs.ParseJSON(buf)
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
					return 2
				}
				s.Push(json.JsonType.NewValue(s, g))
				s.Push(LNil)
				return 2
			})

	//region Client
	chkClient := func(s *LState) *Client {
		ud := s.CheckUserData(1)
		if v, ok := ud.Value.(*Client); ok {
			return v
		}
		s.ArgError(1, "http.Client expected")
		return nil
	}
	ClientType = glu.NewType("Client", false, `client.new(timeoutSeconds int)`,
		func(s *LState) interface{} {
			return NewClient(time.Duration(s.CheckInt(1)) * time.Second)
		}).
		AddMethod("get", `Client:get(url string)(Res?,error?) ==>perform GET request`,
			func(s *LState) int {
				c := chkClient(s)
				res, err := c.Get(s.CheckString(2))
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(ResType.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethod("post", `Client:post(url,contentType, data string)(Res?,error?) ==>perform POST request`,
			func(s *LState) int {
				c := chkClient(s)
				res, err := c.Post(s.CheckString(2), s.CheckString(3), s.CheckString(4))
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(ResType.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethod("head", `Client:head(url string)(Res?,error?) ==>perform HEAD request`,
			func(s *LState) int {
				c := chkClient(s)
				res, err := c.Head(s.CheckString(2))
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(ResType.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethod("form", `Client:form(url string, form map[string][]string)(Res?,error?) ==>perform POST form request`,
			func(s *LState) int {
				c := chkClient(s)
				h := tableToMultiMap(s, 3)
				res, err := c.Form(s.CheckString(2), h)
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(ResType.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethod("do", `Client:do(method, url, data string, header map[string]string)(Res?,error?) ==>perform  request`,
			func(s *LState) int {
				c := chkClient(s)
				res, err := c.Do(
					s.CheckString(2),
					s.CheckString(3),
					s.CheckString(4),
					tableToMap(s, 5),
				)
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(ResType.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethod("doJson", `Client:doJson(method, url string,data json.Json, header map[string]string)(Res?,error?) ==>perform request`,
			func(s *LState) int {
				c := chkClient(s)
				m := tableToMap(s, 5)
				m["Content-Type"] = "application/json"
				res, err := c.Do(
					s.CheckString(2),
					s.CheckString(3),
					json.JsonTypeCheck(s, 4).String(),
					m,
				)
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(ResType.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			})
	//endregion
	HttpModule.AddModule(CtxType)
	HttpModule.AddModule(ServerType)
	HttpModule.AddModule(ClientType)
	HttpModule.AddModule(ResType)
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
		c.SetStatus(500)
		c.SendString(err.Error())
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
func tableToMultiMap(s *LState, n int) (r map[string][]string) {
	v := s.Get(n)
	if v.Type() == LTNil {
		return
	}
	t := s.CheckTable(n)
	if t == nil {
		return
	}
	r = make(map[string][]string)
	t.ForEach(func(k LValue, v LValue) {
		r[k.String()] = valueToSlice(v)
	})
	return
}
func tableToMap(s *LState, n int) (r map[string]string) {
	v := s.Get(n)
	if v.Type() == LTNil {
		return
	}
	t := s.CheckTable(n)
	if t == nil {
		return
	}
	r = make(map[string]string)
	t.ForEach(func(k LValue, v LValue) {
		r[k.String()] = v.String()
	})
	return
}
func valueToSlice(t LValue) (r []string) {
	switch t.Type() {
	case LTNil:
		return nil
	case LTString, LTBool, LTNumber:
		r = append(r, t.String())
		return
	case LTTable:
		t.(*LTable).ForEach(func(k LValue, v LValue) {
			r = append(r, v.String())
		})
	}
	return
}
