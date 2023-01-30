package http

import (
	"github.com/ZenLiuCN/glu"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type echo int

func (e echo) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	_, _ = writer.Write([]byte(request.Method))
}

func init() {
	go func() {
		_ = http.ListenAndServe(":80", echo(1))
	}()
}
func TestNewClient(t *testing.T) {
	if err := glu.ExecuteCode(`
	local res,err=require('http').Client.new(5):get('http://127.0.0.1')
	if err==nil then
		local txt=res:body()
		print(txt)
	else
		print(err)
	end
	`, 0, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
}
func TestClient_Form(t *testing.T) {
	c := NewClient(time.Second)
	frm := url.Values{}
	_, err := c.Form("http://127.0.0.1", frm)
	if err != nil {
		t.Fatal(err)
	}
}
func TestClient_Get(t *testing.T) {
	_, err := NewClient(time.Second).Get("http://127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
}
func TestClient_Post(t *testing.T) {
	_, err := NewClient(time.Second).Post("http://127.0.0.1", "application/json", `{}`)
	if err != nil {
		t.Fatal(err)
	}
}
func TestClient_Head(t *testing.T) {
	_, err := NewClient(time.Second).Head("http://127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
}
func TestClient_Do(t *testing.T) {
	_, err := NewClient(time.Second).Do("HEAD", "http://127.0.0.1", "", nil)
	if err != nil {
		t.Fatal(err)
	}
}
func TestClient_DoWithData(t *testing.T) {
	_, err := NewClient(time.Second).Do("HEAD", "http://127.0.0.1", "123", nil)
	if err != nil {
		t.Fatal(err)
	}
}
func TestClient_DoWithHeader(t *testing.T) {
	_, err := NewClient(time.Second).Do("HEAD", "http://127.0.0.1", "123", map[string]string{
		"content-type": "text/plain",
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestClient_DoInvalidMethod(t *testing.T) {
	_, err := NewClient(time.Second).Do("中", "http://127.0.0.1", "", nil)
	if err == nil {
		t.Fatal()
	}
	_, err = NewClient(time.Second).Do("中", "http://127.0.0.1", "123", nil)
	if err == nil {
		t.Fatal()
	}
}
