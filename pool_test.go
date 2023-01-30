package glu

import (
	"fmt"
	. "github.com/chzyer/test"
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

func TestGluI1(t *testing.T) {

	vmPool := CreatePool()
	vm := vmPool.Get()
	NotNil(vm)
	vmaddr := fmt.Sprintf("%p", vm)

	Equal(LNil, vm.GetGlobal("__SOME_KEY__"))           // Passed
	Success(vm.DoString("assert(__SOME_KEY__ == nil)")) // Passed, __SOME_KEY__ is nil in and out of lua
	vm.SetGlobal("__SOME_KEY__", LString("ZZZZZZ"))
	Equal(LString("ZZZZZZ"), vm.GetGlobal("__SOME_KEY__"))   // Passed
	Success(vm.DoString("assert(__SOME_KEY__ == 'ZZZZZZ')")) // Passed, __SOME_KEY__ is same in and out of lua

	vmPool.Put(vm) // Recycle the vm
	vm = vmPool.Get()
	vmaddr2 := fmt.Sprintf("%p", vm)
	Equal(vmaddr, vmaddr2) // vm is actually recycled

	// Something strange happened....
	// Global is not reset after recycle
	Equal(LNil, vm.GetGlobal("__SOME_KEY__")) // Assertion failed, Value still exists out of lua
	// But I cannot find it in lua, __SOME_KEY__ is nil
	Success(vm.DoString("assert(__SOME_KEY__ == nil)")) // Passed, Nothing in lua

	// What if I set it again?
	vm.SetGlobal("__SOME_KEY__", LString("YYYYYY"))
	Equal(LString("YYYYYY"), vm.GetGlobal("__SOME_KEY__"))   // Passed
	Success(vm.DoString("assert(__SOME_KEY__ == 'YYYYYY')")) // Passed
}
