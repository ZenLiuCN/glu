package http

import (
	"github.com/Jeffail/gabs/v2"
	"github.com/ZenLiuCN/glu/v3"
	"net/http"
	"testing"
	"time"
)

func TestHttpHelp(t *testing.T) {
	s := glu.Get()
	err := s.DoString(
		//language=lua
		`
		local http=require('http')
		local json=require('json')
		print(http.help('?'))
		print(http.Server.help('?'))
		for word in string.gmatch(http.Server.help(), '([^,]+)') do
			print(http.Server.help(word))
		end
		print(http.CTX.help('?'))
		for word in string.gmatch(http.CTX.help(), '([^,]+)') do
			print(http.CTX.help(word))
		end
	`)
	if err != nil {
		t.Fatal(err)
	}
}
func TestHttpServer(t *testing.T) {
	const u = "http://127.0.0.1:8033/"
	go func() {
		time.Sleep(time.Second * 5)
		for _, server := range POOL {
			_, _ = server.Stop(time.Second)
		}
	}()
	go func() {
		time.Sleep(time.Second * 1)
		c := NewClient(time.Second)
		req := func(r *http.Response) {
			g, err := gabs.ParseJSONBuffer(r.Body)
			if err != nil {
				t.Error(err)
				return
			}
			if g.String() != `{"a":1}` {
				t.Error("result: ", g.String())
				return
			}
		}
		if r, err := c.Get(u + `?p={"a":1}`); err != nil {
			t.Error(err)
			return
		} else {
			req(r)
		}
		if _, err := c.Head(u + `?p={"a":1}`); err != nil {
			t.Error(err)
			return
		}
		t.Log("success")
	}()
	if err := glu.ExecuteCode(
		//language=lua
		`
		local http=require('http')
		local json=require('json')
		local server=http.Server.new(':8033')
		function sj(c)
			c:sendJson(require('json').JSON.new(c:query('p')))
		end
		function ss(c)
			c:sendString(c:query('p'))
		end
		server:get('/',sj)
		server:head('/',ss)
		server:post('/',ss)
		server:put('/',ss)
		server:post('/',ss)
		server:start(false)
	    while (server:running()~=true) do	end
		print('start watch')
		while (server:running()) do	end
		server:release()
		print('closed')
	`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
	if len(POOL) != 0 {
		t.Fail()
	}
}
