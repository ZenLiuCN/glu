package http

import (
	"github.com/Jeffail/gabs/v2"
	"glu"
	"net/http"
	"testing"
	"time"
)

func TestHttpHelp(t *testing.T) {
	s := glu.Get()
	err := s.DoString(`
		local http=require('http')
		local json=require('json')
		print(http.Help('?'))
		print(http.Server.Help('?'))
		for word in string.gmatch(http.Server.Help(), '([^,]+)') do
			print(http.Server.Help(word))
		end
		print(http.Ctx.Help('?'))
		for word in string.gmatch(http.Ctx.Help(), '([^,]+)') do
			print(http.Ctx.Help(word))
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
		for _, server := range ServerPool {
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

	}()
	if err := glu.ExecuteCode(`
		local http=require('http')
		local json=require('json')
		local server=http.Server.new(':8033')
		local sj=chunk("local c=... c:sendJson(require('json').Json.new(c:query('p')))",'handler')
		local ss=chunk("local c=...	c:sendString(c:query('p'))",'ss')
		server:get('/',sj)
		server:head('/',ss)
		server:post('/',ss)
		server:put('/',ss)
		server:post('/',ss)
		server:start(false)
	    while (server:running()~=true) do	end
		-- print('start watch')
		while (server:running()) do	end
		server:release()
	`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
	if len(ServerPool) != 0 {
		t.Fail()
	}
}
