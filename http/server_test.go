package http

import (
	"glu"
	"testing"
)

func TestMuxHttp(t *testing.T) {
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
func TestServe(t *testing.T) {
	s := glu.Get()
	err := s.DoString(`
		local http=require('http')
		local json=require('json')
		local server=http.Server.new(':8081')
		server:get('/',[[
			local c=...
			local json=require('json')
			c:send(json.Json.new(c:query('p')))
		]])
		server:start(false)
		while (true) do	end
	`)
	if err != nil {
		t.Fatal(err)
	}
}
