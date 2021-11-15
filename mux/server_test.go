package mux

import (
	"gua"
	"testing"
)

func TestMuxHttp(t *testing.T) {
	gua.Auto = true
	s := gua.Get()
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
