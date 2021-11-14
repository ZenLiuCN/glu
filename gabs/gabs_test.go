package gabs

import (
	"gua"
	"testing"
)

func TestJsonNew(t *testing.T) {
	s := gua.Get()
	err := s.DoString(`
local json=require('json')
print(json.Help())
print(json.json.Help())
print(json.json.Help('string'))
`)
	if err != nil {
		t.Fatal(err)
	}
}
