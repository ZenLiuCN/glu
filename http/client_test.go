package http

import (
	"glu"
	"testing"
)

func TestClient(t *testing.T) {
	s := glu.Get()
	err := s.DoString(`
	local res,err=require('http').Client.new(5):get('http://github.com')
	local txt=res:body()
	print(err)
	print(txt)
	`)
	if err != nil {
		t.Fatal(err)
	}
}
