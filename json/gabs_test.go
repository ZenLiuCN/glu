package json

import (
	"glu"
	"testing"
)

func TestJsonNew(t *testing.T) {
	s := glu.Get()
	err := s.DoString(`
local json=require('json')
local newJson=json.Json.new
print('support =>'..json.Help())
print(json.Help('?'))
print(json.Help('from'))
print(json.from('a'):json())
print(json.from(12.5):json())
print(json.from(true):json())
print(json.from({1,2,3}):json())
print('support =>'..json.Json.Help())
local m=newJson('{"va":1,"boo":true,"ar":[1,2,"string"],"ob":{"1":1}}')
print(m:json())
print(json.Json.Help('json'))
print('default: \n'..m:json())
print('pretty: \n'..m:json(true))
print('pretty and indent by space: \n'..m:json(true,' '))
print(json.Json.Help('path'))
print(m:path('boo'):json())
print(json.Json.Help('at'))
print(m:at('boo'):json())
print(m:at('ar'):json())
print(m:at('ar'):at(2):json())
print(json.Json.Help('type'))
print(m:at('ar'):at(2):type())
print(json.Json.Help('set'))
m:set('ar.2',nil)
print(m:json(true))
print(json.Json.Help('append'))
m:append('ar',5)
print(m:json())
print(json.Json.Help('array'))
print(m:path('ar'):json()..'=>'..tostring(m:array('ar')))
print(m:path('va'):json()..'=>'..tostring(m:array('va')))
print(json.Json.Help('object'))
print(m:path('ar'):json()..'=>'..tostring(m:object('ar')))
print(m:path('ob'):json()..'=>'..tostring(m:object('ob')))

`)
	if err != nil {
		t.Fatal(err)
	}
}
