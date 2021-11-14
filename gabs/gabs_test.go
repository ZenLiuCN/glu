package gabs

import (
	"gua"
	"testing"
)

func TestJsonNew(t *testing.T) {
	s := gua.Get()
	err := s.DoString(`
local json=require('json')
local newJson=json.json.new
print(json.Help())
print(json.json.Help())
print(json.json.Help('string'))
print(json.json.Help('string1'))
print(json.json.Help('json'))
print(json.json.new():string1())
print(newJson():json())
`)
	if err != nil {
		t.Fatal(err)
	}
}
