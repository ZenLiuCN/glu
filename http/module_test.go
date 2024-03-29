package http

import (
	"bytes"
	"github.com/ZenLiuCN/glu/v3"
	"github.com/yuin/gopher-lua"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestModuleHelp(t *testing.T) {
	if err := glu.ExecuteCode(
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
	`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
func TestModuleToSlice(t *testing.T) {
	s := glu.Get()
	defer glu.Put(s)
	tab := s.NewTable()
	tab.RawSetInt(1, lua.LString("1"))
	tab.RawSetInt(2, lua.LString("2"))
	tab.RawSetInt(3, lua.LString("3"))
	s.Push(tab)
	sl := tableToSlice(s.LState, 1)
	if sl[0] != "1" || sl[1] != "2" || sl[2] != "3" {
		t.Fatal()
	}
	s.Pop(1)
	sl = tableToSlice(s.LState, 1)
	if sl != nil {
		t.Fatal()
	}
	sl = valueToSlice(tab)
	if sl[0] != "1" || sl[1] != "2" || sl[2] != "3" {
		t.Fatal()
	}
	if valueToSlice(lua.LNil) != nil {
		t.Fatal()
	}
	if valueToSlice(lua.LString("1"))[0] != "1" {
		t.Fatal()
	}
	if valueToSlice(lua.LNumber(1))[0] != "1" {
		t.Fatal()
	}
	if valueToSlice(lua.LTrue)[0] != "true" {
		t.Fatal()
	}
}
func TestModuleToMultiMap(t *testing.T) {
	s := glu.Get()
	defer glu.Put(s)
	tab := s.NewTable()
	t1 := s.NewTable()
	t1.RawSetInt(1, lua.LNumber(3))
	t1.RawSetInt(2, lua.LString("b"))
	tab.RawSetString("1", lua.LString("1"))
	tab.RawSetString("2", lua.LString("2"))
	tab.RawSetString("3", t1)
	s.Push(tab)
	defer s.Pop(1)
	sl := tableToMultiMap(s.LState, 1)
	if sl["1"][0] != "1" || sl["2"][0] != "2" || sl["3"][0] != "3" || sl["3"][1] != "b" {
		t.Fatal()
	}
}
func TestModuleToMap(t *testing.T) {
	s := glu.Get()
	defer glu.Put(s)
	tab := s.NewTable()
	tab.RawSetString("1", lua.LString("1"))
	tab.RawSetString("2", lua.LString("2"))
	s.Push(tab)
	sl := tableToMap(s.LState, 1)
	if sl["1"] != "1" || sl["2"] != "2" {
		t.Fatal()
	}
	s.Pop(1)
	sl = tableToMap(s.LState, 1)
	if sl != nil {
		t.Fatal()
	}
}
func TestModuleExecuteHandler(t *testing.T) {
	s := glu.Get()
	defer glu.Put(s)
	err := s.DoString(
		//language=lua
		`
function x(a) 
	assert(a:header('1')~=nil)
	print('call 01')
	print('header:',a:header('1'))
end 
return x`)
	if err != nil {
		t.Fatal(err)
	}
	fn := s.Get(1).(*lua.LFunction)
	s.Pop(1)
	x := &Ctx{
		Request: &http.Request{
			Header: make(http.Header),
		},
	}
	x.Request.Header["1"] = []string{"1", "2"}
	executeHandler(fn, x)
	err = s.DoString(
		//language=lua
		`
function x(a) 
	print('call 02')
	assert(a:header('1')==nil)
end 
return x`)
	if err != nil {
		t.Fatal(err)
	}
	fn = s.Get(1).(*lua.LFunction)
	s.Pop(1)
	t.Log("test fn2")
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("should have error")
		} else {
			t.Log(r)
		}
	}()
	executeHandler(fn, x)
}
func TestModuleCtx(t *testing.T) {
	s := glu.Get()
	defer glu.Put(s)
	err := s.DoString(
		//language=lua
		`
function x(a)
		assert(a:header('1')=='')
		assert(a:header('a')=='b')
		assert(a:vars('a')=='')
		assert(a:query('p')=='1')
		assert(a:method()=='PUT')
		assert(a:body():json()=='{"1":2}')
		a:setHeader('v','a')
		a:status(500)
		a:sendString("1")
end
return x
`)
	if err != nil {
		t.Fatal(err)
	}
	c := s.Get(1).(*lua.LFunction)
	s.Pop(1)
	i, _ := url.Parse("http://127.0.0.1/p?p=1")
	x := &Ctx{
		Request: &http.Request{
			Header: map[string][]string{
				"A": {"b"},
			},
			Method: http.MethodPut,
			URL:    i,
			Body:   io.NopCloser(strings.NewReader(`{"1":2}`)),
		},
		ResponseWriter: new(httptest.ResponseRecorder),
	}
	w := x.ResponseWriter.(*httptest.ResponseRecorder)
	w.Body = new(bytes.Buffer)
	executeHandler(c, x)
	if w.Header().Get("v") != "a" {
		t.Fail()
	}
	if string(w.Body.Bytes()) != "1" {
		t.Fatal()
	}
	if w.Code != 500 {
		t.Fatal()
	}
}
func TestModuleRes(t *testing.T) {
	x := glu.Get()
	defer glu.Put(x)
	r := &http.Response{
		Status:           "200 OK",
		StatusCode:       200,
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           map[string][]string{"V": {"1"}},
		Body:             io.NopCloser(bytes.NewReader([]byte(`{"a":1}`))),
		ContentLength:    100,
		TransferEncoding: nil,
		Close:            false,
		Uncompressed:     false,
		Trailer:          nil,
		Request:          nil,
		TLS:              nil,
	}
	err := glu.ExecuteCode(`
local a=... 
assert(a~=nil)
assert(a~=nil)
assert(a:statusCode()==200)
assert(a:status()=='200 OK')
assert(a:size()==100)
assert(a:header()['V']=="1")
assert(a:bodyJson():json()=='{"a":1}')
`, 1, 0, func(s *glu.Vm) error {
		s.Push(RESPONSE.NewValue(s.LState, r))
		return nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	r = &http.Response{
		Status:           "200 OK",
		StatusCode:       200,
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           map[string][]string{"V": {"1"}},
		Body:             io.NopCloser(bytes.NewReader([]byte(`{"a":1}`))),
		ContentLength:    100,
		TransferEncoding: nil,
		Close:            false,
		Uncompressed:     false,
		Trailer:          nil,
		Request:          nil,
		TLS:              nil,
	}
	err = glu.ExecuteCode(`
local a=... 
assert(a:body()=='{"a":1}')
`, 1, 0, func(s *glu.Vm) error {
		s.Push(RESPONSE.NewValue(s.LState, r))
		return nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}
