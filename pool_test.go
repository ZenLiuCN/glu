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
	snap := func() (r []string) {
		println(l.Env.MaxN(), l.Env.Len())
		l.Env.ForEach(func(k LValue, v LValue) {
			r = append(r, k.String())
		})
		return r
	}
	r0 := snap()
	err := l.DoString(`
		a=1 print(tostring(a).."1+")
	`)
	if err != nil {
		return
	}
	r1 := snap()
	Put(l)
	r2 := snap()
l0:
	for _, s := range r1 {
		for _, s2 := range r0 {
			if s2 == s {
				continue l0
			}
		}
		println("pollution: ", s)
	}

l2:
	for _, s := range r2 {
		for _, s2 := range r0 {
			if s2 == s {
				continue l2
			}
		}
		println("pollution: ", s)
	}
}
