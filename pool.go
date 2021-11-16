package glu

import (
	. "github.com/yuin/gopher-lua"
	"sync"
)

var (
	//Option LState configuration
	Option   Options = Options{}
	PoolSize         = 4
	pool     *StatePool
)

//MakePool manual create statePool , when need to change Option, should invoke once before use Get and Put
func MakePool() {
	pool = CreatePool()
}

//Get LState from statePool
func Get() *LState {
	if pool == nil {
		MakePool()
	}
	return pool.Get()
}

//Put LState back to statePool
func Put(s *LState) {
	if pool == nil {
		MakePool()
	}
	pool.Put(s)
}

var (
	//Auto if true, will auto-load modules in Registry
	Auto = true
)

//region Pool

//StatePool threadsafe LState Pool
type StatePool struct {
	m     sync.Mutex
	saved []*LState //TODO replace with more effective structure
	ctor  func() *LState
}

//CreatePoolWith create pool with user defined constructor
//
//**Note** GluModule will auto registered
func CreatePoolWith(ctor func() *LState) *StatePool {
	return &StatePool{saved: make([]*LState, 0, PoolSize), ctor: ctor}
}
func CreatePool() *StatePool {
	return &StatePool{saved: make([]*LState, 0, PoolSize)}
}

func (pl *StatePool) Get() *LState {
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

func (pl *StatePool) new() *LState {
	if pl.ctor != nil {
		l := pl.ctor()
		GluModule.PreLoad(l)
		return l
	}
	L := NewState(Option)
	configurer(L)
	return L
}

func (pl *StatePool) Put(L *LState) {
	if L.IsClosed() {
		return
	}
	// reset stack
	L.Pop(L.GetTop())
	pl.m.Lock()
	defer pl.m.Unlock()
	pl.saved = append(pl.saved, L)
}

func (pl *StatePool) Shutdown() {
	for _, L := range pl.saved {
		L.Close()
	}
}

//endregion
func configurer(l *LState) {
	GluModule.PreLoad(l)
	if Auto {
		for _, module := range Registry {
			module.PreLoad(l)
		}
	}
}
