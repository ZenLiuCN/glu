package gua

import (
	. "github.com/yuin/gopher-lua"
	"sync"
)

var (
	Option Options = Options{}
	pool   *p
)

func MakePool() {
	pool = create()
}
func Get() *LState {
	if pool == nil {
		MakePool()
	}
	return pool.get()
}
func Put(s *LState) {
	if pool == nil {
		MakePool()
	}
	pool.put(s)
}

//region Pool
type p struct {
	p sync.Pool
}

func create() *p {
	return &p{sync.Pool{
		New: func() interface{} {
			L := NewState(Option)
			configurer(L)
			return L
		},
	}}
}
func (x *p) get() *LState {
	return x.p.Get().(*LState)
}
func (x *p) put(s *LState) {
	x.p.Put(s)
}

//endregion
func configurer(l *LState) {
	for _, module := range Registry {
		module.PreLoad(l)
	}
}
