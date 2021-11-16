package glu

import (
	lua "github.com/yuin/gopher-lua"
	"testing"
)

var (
	chunk1 *lua.FunctionProto
	chunk2 *lua.FunctionProto
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
