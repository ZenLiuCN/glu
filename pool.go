package glu

import (
	. "github.com/yuin/gopher-lua"
	"sync"
)

var (
	//Option LState configuration
	Option   Options = Options{}
	PoolSize         = 4
	pool     *statePool
)

//MakePool manual create statePool , when need to change Option, should invoke once before use Get and Put
func MakePool() {
	pool = create()
}

//Get LState from statePool
func Get() *LState {
	if pool == nil {
		MakePool()
	}
	return pool.get()
}

//Put LState back to statePool
func Put(s *LState) {
	if pool == nil {
		MakePool()
	}
	pool.put(s)
}

var (
	//Auto if true, will auto-load modules in Registry
	Auto = true
)

//region Pool
type statePool struct {
	m     sync.Mutex
	saved []*LState
}

func create() *statePool {
	return &statePool{saved: make([]*LState, 0, PoolSize)}
}

func (pl *statePool) get() *LState {
	pl.m.Lock()
	defer pl.m.Unlock()
	n := len(pl.saved)
	if n == 0 {
		return pl.new()
	}
	x := pl.saved[n-1]
	pl.saved = pl.saved[0 : n-1]
	return x
}

func (pl *statePool) new() *LState {
	L := NewState(Option)
	configurer(L)
	return L
}

func (pl *statePool) put(L *LState) {
	pl.m.Lock()
	defer pl.m.Unlock()
	pl.saved = append(pl.saved, L)
}

func (pl *statePool) Shutdown() {
	for _, L := range pl.saved {
		L.Close()
	}
}

//endregion
func configurer(l *LState) {
	if Auto {
		for _, module := range Registry {
			module.PreLoad(l)
		}
	}

}
