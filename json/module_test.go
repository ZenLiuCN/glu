package json

import (
	. "github.com/ZenLiuCN/glu/v3"
	"testing"
)

func TestJsonHelp(t *testing.T) {
	if err := ExecuteCode(`print(help())`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := ExecuteCode(`
local json=require('json')
for word in string.gmatch(json.help(), '([^,]+)') do
	print(json.help(word))
end
for word in string.gmatch(json.JSON.help(), '([^,]+)') do
	print(json.JSON.help(word))
end
`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestJsonQuery1(t *testing.T) {
	if ExecuteCode(`
local j=require('json').Json.new()
assert(j:get(true))
`, 0, 0, nil, nil) == nil {
		t.Fatal("should fail")
	}

}
func TestJsonQuery2(t *testing.T) {
	if err := ExecuteCode(
		//language=lua
		`
local json=require('json')
local js='{"a":{"a1":1,"a2":true,"a3":"123"},"b":[1,2,3],"c":"c","d":1,"e":1.2,"f":true,"g":false}'
local j=json.JSON.new(js)
assert(j:json()==js)
assert(j:get('a'):type()==5)
assert(j:isObject('a'))
assert(j:isArray('a')==false)
assert(j:isArray('x')==false)
assert(j:isObject('x')==false)
assert(j:get('b'):type()==4)
assert(j:isArray('b'))
assert(j:isObject('b')==false)
assert(j:get('f'):type()==3)
assert(j:get('d'):type()==2)
assert(j:get('c'):type()==1)
assert(j:get('b'):json()=='[1,2,3]')
assert(j:get('a'):get("a2"):json()=='true')
assert(j:get('a'):get("a5")==nil)
assert(j:get('b'):get(1):json()=='2')
assert(j:get('b'):get(4)==nil)
assert(j:exists('b')==true)
assert(j:exists('h')==false)
assert(j:path('b.1'):json()=='2')

assert(j:path('a'):bool('a2'))
assert(j:path('a'):bool('a3')==false)
assert(j:bool('g')==false)
assert(j:bool('h')==false)
assert(j:path('a'):string('a3')=='123')
assert(j:path('a'):string('a4')==nil)
assert(j:path('a'):string('a2')==nil)
assert(j:path('a'):number('a3')==nil)
assert(j:path('a'):number('a8')==nil)
assert(j:path('a'):number('a1')==1)
assert(j:path('x')==nil)

`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
func TestJsonAppend1(t *testing.T) {

	if err := ExecuteCode(
		//language=lua
		`
local json=require('json')
local j=json.JSON.new('{"a":1,"b":{"c":1}}')
assert(j:append(1)==nil)
assert(j:append("b",1)==nil)
`, 0, 0, nil, nil); err == nil {
		t.Fatal("should fail")
	}

}
func TestJsonAppend2(t *testing.T) {
	if err := ExecuteCode(
		//language=lua
		`
local json=require('json')
local j=json.JSON.new('{"a":1,"b":{"c":1}}')
assert(j:append("b",1)==nil)
`, 0, 0, nil, nil); err == nil {
		t.Fatal("should fail")
	}
}
func TestJsonAppend3(t *testing.T) {
	if err := ExecuteCode(
		//language=lua
		`
local json=require('json')
local j=json.JSON.new('{"a":1}')
assert(j:append("a",1)==nil)
assert(j:append("a",1)==nil) 
assert(j:append("a",true)==nil) 
assert(j:append("a",1.2)==nil) 
assert(j:append("a","a")==nil) 
assert(j:append("a",json.of({["a"]=1}))==nil) 
assert(j:append("a",nil)==nil) print(j:json())
assert(j:append("b",nil)==nil) print(j:json())
assert(j:append("b",1)==nil) print(j:json())
assert(j:append("b",2)==nil) print(j:json())
assert(j:append("b",nil)==nil) print(j:json())
assert(j:set("c",json.JSON.new())==nil) print(j:json())
assert(j:append("c",1)==nil) print(j:json())
assert(j:size("a")==6)
`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
func TestJsonAppend4(t *testing.T) {
	if err := ExecuteCode(
		//language=lua
		`
local json=require('json')
local j=json.JSON.new()
assert(j:append(1)==nil)
assert(j:append(1)==nil) 
assert(j:append(true)==nil) 
assert(j:append(1.2)==nil) 
assert(j:append("a")==nil) 
assert(j:append(json.of({["a"]=1}))==nil) 
assert(j:append(nil)==nil) print(j:json())
assert(j:size()==5)
`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestJsonSet(t *testing.T) {
	if err := ExecuteCode(
		//language=lua
		`
local json=require('json')
local j=json.JSON.new('{"a":1}')
assert(j:append(1)==nil)
`, 0, 0, nil, nil); err == nil {
		t.Fatal("should fail")
	}
	if err := ExecuteCode(
		//language=lua
		`
local json=require('json')
local j=json.JSON.new('{"a":1}')
assert(j:set("a",1)==nil)
assert(j:set("a",1)==nil) 
assert(j:set("a",true)==nil) 
assert(j:set("a",1.2)==nil) 
assert(j:set("a","a")==nil) 
assert(j:set("a",json.of({["a"]=1}))==nil) 
assert(j:set("a",nil)==nil) print(j:json())
assert(j:set(nil)==nil) print(j:json())
assert(j:set('',nil)==nil) print(j:json())
assert(j:set("a",12)==nil) print(j:json())
assert(j:size()==1)
assert(j:size("a")==1)
assert(j:size("c")==nil)
assert(j:set("",12)==nil) print(j:json())

`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
func TestJsonCreate(t *testing.T) {
	if err := ExecuteCode(
		//language=lua
		`
local json=require('json')
local j=json.JSON.new({1,2,3})
`, 0, 0, nil, nil); err == nil {
		t.Fatal("should fail")
	}
	if err := ExecuteCode(
		//language=lua
		`
local json=require('json')
local Json=json.JSON
local j=json.JSON.new()
assert(j:json()=='{}')
assert(Json.new():json()=='{}')
assert(Json.new('{"a":1}'):json()=='{"a":1}')
assert(json.of({1,2,3}):json()=='[1,2,3]')
local x=json.of({['a']=1,['b']=2,['c']={['1']=1,["a"]="s",["c"]=true}})
assert(x:json()=='{"a":1,"b":2,"c":{"1":1,"a":"s","c":true}}')
assert(json.of(1):json()=='1')
assert(json.of("1"):json()=='"1"')
assert(json.of(true):json()=='true')
assert(json.stringify(json.of(true))=='true')
print(tostring(json.of(true)))
assert(tostring(json.of(true))=='true')
`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}

}

func TestJsonRaw(t *testing.T) {
	if err := ExecuteCode(
		//language=lua
		`
local json=require('json')
local j=json.of({1,2,3})
local t=j:raw()
assert(t[1]==1)
assert(t[2]==2)
assert(t[3]==3)

j=json.of({['a']=1,['b']=2,['c']=3})
t=j:raw()
print(json.of(t):json())
assert(t['a']==1)
assert(t['b']==2)
assert(t['c']==3)
`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
