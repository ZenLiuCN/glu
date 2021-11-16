package glu

import (
	. "github.com/yuin/gopher-lua"
	"testing"
)

var (
	chunk1 *FunctionProto
	chunk2 *FunctionProto
)

func init() {
	var err error
	chunk1, err = CompileChunk(`local a=1+1`, `bench`)
	if err != nil {
		panic(err)
	}
	chunk2, err = CompileChunk(`local a=1+1; assert(a~=2)`, `bench`)
	if err != nil {
		panic(err)
	}

}
func BenchmarkPoolWithoutClose(b *testing.B) {
	for i := 0; i < b.N; i++ {
		x := Get()
		if i%2 == 0 {
			x.Push(x.NewFunctionFromProto(chunk1))
		} else {
			x.Push(x.NewFunctionFromProto(chunk2))
		}
		err := x.PCall(0, 0, nil)
		if err != nil {
			Put(x)
			continue
		}
		Put(x)
	}
}
func BenchmarkPoolWithClose(b *testing.B) {
	for i := 0; i < b.N; i++ {
		x := Get()
		if i%2 == 0 {
			x.Push(x.NewFunctionFromProto(chunk1))
		} else {
			x.Push(x.NewFunctionFromProto(chunk2))
		}
		err := x.PCall(0, 0, nil)
		if err != nil {
			x.Close()
			continue
		}
		Put(x)
	}
}

func TestStateTrace(t *testing.T) {
	l := Get()
	err := l.DoString(`
		a=1 print(tostring(a).."1+")
	`)
	if err != nil {
		return
	}
	println("before put: ", l.Polluted())
	Put(l)
	println("after put: ", l.Polluted())
}
