package http

import (
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/ZenLiuCN/fn"
	. "github.com/ZenLiuCN/glu/v3"
	"github.com/ZenLiuCN/glu/v3/json"
	. "github.com/yuin/gopher-lua"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	CTX      Type[*Ctx]
	SERVER   Type[*Server]
	RESPONSE Type[*http.Response]
	CLIENT   Type[*Client]
	MODULE   Module
	POOL     map[int64]*Server
	CLIENTS  map[int64]*Client
)

func init() {
	POOL = make(map[int64]*Server, 4)
	CLIENTS = make(map[int64]*Client, 4)
	MODULE = NewModule("http", `http module built on net/http gorilla/mux, requires json module. minimal sample as below:
local http=require('http')
local json=require('json')
local server=http.Server.new(':8081')
local function rec(ctx)
	ctx:sendString('ok')
end
server:get('/',rec)
server:start(false)
while (true) do	end
`, true)
	//region Ctx

	CTX = NewTypeCast(func(a any) (v *Ctx, ok bool) { v, ok = a.(*Ctx); return }, "CTX", `http request context`, false, "", nil).
		AddMethodCast("vars", `(string)string 	 fetch path variable`, func(s *LState, v *Ctx) int {
			s.Push(LString(v.Vars(s.CheckString(2))))
			return 1
		}).
		AddMethodCast("header", `(string)string 	 fetch request header`,
			func(s *LState, v *Ctx) int {
				s.Push(LString(v.Header(s.CheckString(2))))
				return 1
			}).
		AddMethodCast("query", `(string)string 	 fetch request query parameter`,
			func(s *LState, v *Ctx) int {
				s.Push(LString(v.Query(s.CheckString(2))))
				return 1
			}).
		AddMethodCast("method", `()string 	 fetch request method`,
			func(s *LState, v *Ctx) int {
				s.Push(LString(v.Method()))
				return 1
			}).
		AddMethodCast("body", `()JSON 	 fetch request body`,
			func(s *LState, v *Ctx) int {
				body, err := v.Body()
				if err != nil {
					s.RaiseError("fetch body error %s", err)
				}
				return json.JSON.New(s, body)
			}).
		AddMethodUserData("setHeader", `(string,string)CTX 	 chain method set output header`, func(s *LState, u *LUserData) int {
			key := s.CheckString(2)
			value := s.CheckString(3)
			v := CTX.CheckUserData(u, s)
			v.SetHeader(key, value)
			s.Push(u)
			return 1
		}).
		AddMethodCast("status", `(int) 	 send http status,this will end process`, func(s *LState, v *Ctx) int {
			v.SetStatus(s.CheckInt(2))
			return 0
		}).
		AddMethodCast("sendJson", `(JSON) 	 send json body,this will end process`, func(s *LState, v *Ctx) int {
			v.SendJson(json.JSON.Check(s, 2))
			return 0
		}).
		AddMethodCast("sendString", `(string) 	 send text body,this will end process`, func(s *LState, v *Ctx) int {
			g := s.CheckString(2)
			v.SendString(g)
			return 0
		}).
		AddMethodCast("sendFile", `(string) 	 send file,this will end process`, func(s *LState, v *Ctx) int {
			err := v.SendFile(s.CheckString(2))
			if err != nil {
				s.RaiseError("send file %s", err)
			}
			return 0
		})
	//endregion
	//region Server

	SERVER = NewTypeCast(func(a any) (v *Server, ok bool) { v, ok = a.(*Server); return }, "Server", `Http Server`, false, `(addr string)`,
		func(s *LState) *Server {
			srv := NewServer(s.CheckString(1), func(s string) {
				log.Println(s)
			})
			POOL[srv.ID] = srv
			return srv
		}).
		AddMethodCast("stop", `(seconds int) 	 stop server graceful`,
			func(s *LState, v *Server) int {
				s.CheckType(2, LTNumber)
				err, _ := v.Stop(time.Second * time.Duration(s.ToInt(2)))
				if err != nil {
					s.RaiseError("stop http.Server fail %s", err)
				}
				return 0
			}).
		AddMethodCast("running", `()bool 	 check server is running`,
			func(s *LState, v *Server) int {
				if v.Running() {
					s.Push(LTrue)
				} else {
					s.Push(LFalse)
				}
				return 1
			}).
		AddMethodCast("start", `(
    cors bool,                 		 enable cors or not,default false.
	allowHeader    {string}?,  		 cors config for header allowed.
	allowedMethods {string}?,  		 cors config for methods allowed.
	allowedOrigins {string}?,  		 cors config for origin allowed.
	exposedHeaders {string}?,  		 cors config for header exposed.
	maxAge int?,               		 cors config for maxAge ,maximum 600 seconds.
	allowCredentials bool?,    		 cors config for allowCredentials.
)                              		 start server,should only call once.`,
			func(s *LState, v *Server) int {
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
		AddMethodCast("route", `(path string, handler Function) 	 register handler without method limit.`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				chunk := s.CheckFunction(3)
				v.Route(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethodCast("get", `(path string, handler Function) 	 register handler limit with GET.`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				chunk := s.CheckFunction(3)
				v.Get(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethodCast("post", `(path string, handler Function) 	 register handler limit with POST.`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				chunk := s.CheckFunction(3)
				v.Post(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethodCast("put", `(path string, handler Function) 	 register handler limit with POST.`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				chunk := s.CheckFunction(3)
				v.Put(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethodCast("head", `(path string,handler Function) 	 register handler limit with HEAD.`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				chunk := s.CheckFunction(3)
				v.Head(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethodCast("patch", `(path string,handler Function) 	 register handler limit with PATCH.`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				chunk := s.CheckFunction(3)
				v.Patch(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethodCast("delete", `(path string,handler Function) 	 register handler limit with DELETE.`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				chunk := s.CheckFunction(3)
				v.Delete(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethodCast("connect", `(path string,handler Function) 	 register handler limit with CONNECT.`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				chunk := s.CheckFunction(3)
				v.Connect(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethodCast("options", `(path string,handler Function) 	 register handler limit with OPTIONS.should not use if with cors enable`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				chunk := s.CheckFunction(3)
				v.Options(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethodCast("trace", `(path string,handler Function) 	 register handler limit with TRACE.`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				chunk := s.CheckFunction(3)
				v.Trace(route, func(ctx *Ctx) {
					executeHandler(chunk, ctx)
				})
				return 0
			}).
		AddMethodCast("files", `(path ,prefix, dir string) 	 register path to service with dir`,
			func(s *LState, v *Server) int {
				route := s.CheckString(2)
				pfx := s.CheckString(3)
				file := s.CheckString(4)
				v.File(route, pfx, file)
				return 0
			}).
		AddMethodCast("release", `release() 	 release this server`,
			func(s *LState, v *Server) int {
				if v.Running() {
					_, _ = v.Stop(time.Second)
				}
				delete(POOL, v.ID)
				return 0
			}).
		AddFunc("pool", `()int 	 current pooled server size`,
			func(s *LState) int {
				s.Push(LNumber(len(POOL)))
				return 1
			}).
		AddFunc("poolKeys", `()[number] 	 current pool server keys`,
			func(s *LState) int {
				t := s.NewTable()
				n := 1
				for i := range POOL {
					t.RawSetInt(n, LNumber(i))
					n++
				}
				s.Push(t)
				return 1
			}).
		AddFunc("pooled", `(key number)Server? 	 fetch from pool`,
			func(s *LState) int {
				k := s.CheckInt64(1)
				if v, ok := POOL[k]; ok {
					s.Push(SERVER.NewValue(s, v))
				} else {
					s.Push(LNil)
				}
				return 1
			})
	//endregion

	//region Response

	RESPONSE = NewTypeCast(func(a any) (v *http.Response, ok bool) { v, ok = a.(*http.Response); return }, "Response", `http.Response`, false, ``, nil).
		AddMethodCast("statusCode", `()int`,
			func(s *LState, c *http.Response) int {
				s.Push(LNumber(c.StatusCode))
				return 1
			}).
		AddMethodCast("status", `()string`,
			func(s *LState, c *http.Response) int {
				s.Push(LString(c.Status))
				return 1
			}).
		AddMethodCast("size", `()int 	 content size in bytes`,
			func(s *LState, c *http.Response) int {
				s.Push(LNumber(c.ContentLength))
				return 1
			}).
		AddMethodCast("header", `()map[string]string`,
			func(s *LState, c *http.Response) int {
				t := s.NewTable()
				for k, v := range c.Header {
					t.RawSetString(k, LString(strings.Join(v, ",")))
				}
				s.Push(t)
				return 1
			}).
		AddMethodCast("body", `()string 	 read body as string,body should only read once.`,
			func(s *LState, v *http.Response) int {
				defer v.Body.Close()
				buf, err := io.ReadAll(v.Body)
				if err != nil {
					s.RaiseError("read body %s", err)
					return 0
				}
				s.Push(LString(buf))
				return 1
			}).
		AddMethodCast("bodyJson", `()(JSON?,string?) 	 read body as Json,body should only read once.`,
			func(s *LState, c *http.Response) int {
				defer c.Body.Close()
				buf, err := io.ReadAll(c.Body)
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
				s.Push(json.JSON.NewValue(s, g))
				s.Push(LNil)
				return 2
			})
	//endregion

	//region Client

	CLIENT = NewTypeCast(func(a any) (v *Client, ok bool) { v, ok = a.(*Client); return }, "Client", `http.Client `, false, `(timeoutSeconds int)Client`,
		func(s *LState) *Client {
			c := NewClient(time.Duration(s.CheckInt(1)) * time.Second)
			CLIENTS[c.ID] = c
			return c
		}).
		AddMethodCast("get", `(url string)(Response?,error?) 	perform GET request`,
			func(s *LState, c *Client) int {
				res, err := c.Get(s.CheckString(2))
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(RESPONSE.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethodCast("post", `(url,contentType,data string)(Response?,error?) 	perform POST request`,
			func(s *LState, c *Client) int {
				res, err := c.Post(s.CheckString(2), s.CheckString(3), s.CheckString(4))
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(RESPONSE.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethodCast("head", `(url string)(Response?,error?) 	perform HEAD request`,
			func(s *LState, c *Client) int {
				res, err := c.Head(s.CheckString(2))
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(RESPONSE.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethodCast("form", `(url string, form table)(Response?,error?) 	perform POST form request`,
			func(s *LState, c *Client) int {
				h := tableToMultiMap(s, 3)
				res, err := c.Form(s.CheckString(2), h)
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(RESPONSE.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethodCast("request", `(method, url, data string, header table)(Response?,string?) 	perform  request`,
			func(s *LState, c *Client) int {
				res, err := c.Request(s.CheckString(2), s.CheckString(3), s.CheckString(4), tableToMap(s, 5))
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(RESPONSE.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethodCast("requestJson", `(method, url string,data JSON, header {string:string})(Response?,string?) 	perform request`,
			func(s *LState, c *Client) int {
				m := tableToMap(s, 5)
				m["Content-BaseType"] = "application/json"
				g := json.JSON.Check(s, 4)
				res, err := c.Request(
					s.CheckString(2),
					s.CheckString(3),
					g.String(),
					m,
				)
				if err != nil {
					s.Push(LNil)
					s.Push(LString(err.Error()))
				} else {
					s.Push(RESPONSE.NewValue(s, res))
					s.Push(LNil)
				}
				return 2
			}).
		AddMethodCast("release", `release() 	release this client`,
			func(s *LState, c *Client) int {
				delete(CLIENTS, c.ID)
				return 0
			}).
		AddFunc("pool", `()int 	 current in pool Client size`,
			func(s *LState) int {
				s.Push(LNumber(len(CLIENTS)))
				return 1
			}).
		AddFunc("poolKeys", `(){number} 	 current pool Client keys`,
			func(s *LState) int {
				t := s.NewTable()
				n := 1
				for i := range CLIENTS {
					t.RawSetInt(n, LNumber(i))
					n++
				}
				s.Push(t)
				return 1
			}).
		AddFunc("pooled", `(key number)Server? 	 fetch from pool`,
			func(s *LState) int {
				k := s.CheckInt64(1)
				if v, ok := CLIENTS[k]; ok {
					s.Push(CLIENT.NewValue(s, v))
				} else {
					s.Push(LNil)
				}
				return 1
			})
	//endregion

	fn.Panic(Register(MODULE.AddModule(CTX).AddModule(SERVER).AddModule(CLIENT).AddModule(RESPONSE)))
}
func executeHandler(chunk *LFunction, c *Ctx) {
	if err := ExecuteFunction(chunk, 1, 0, func(s *Vm) error {
		CTX.New(s.LState, c)
		return nil
	}, nil); err != nil {
		c.SetStatus(500)
		c.SendString(err.Error())
		fmt.Printf("handle error %+v : %s", c.URL, err)
		return
	}
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
