package gua

import (
	. "github.com/yuin/gopher-lua"
	"sync"
)

var (
	//Option LState configuration
	Option Options = Options{}
	pool   *p
)

//MakePool manual create pool , when need to change Option, should invoke once before use Get and Put
func MakePool() {
	pool = create()
}

//Get LState from pool
func Get() *LState {
	if pool == nil {
		MakePool()
	}
	return pool.get()
}

//Put LState back to pool
func Put(s *LState) {
	if pool == nil {
		MakePool()
	}
	pool.put(s)
}

var (
	//Auto if true, will auto-load modules in Registry
	Auto = false
)

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
	if Auto {
		for _, module := range Registry {
			module.PreLoad(l)
		}
	}

}
