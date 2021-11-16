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
func Get() *StoredState {
	if pool == nil {
		MakePool()
	}
	return pool.Get()
}

//Put LState back to statePool
func Put(s *StoredState) {
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

//StoredState with Env snapshot to stop Global pollution
type StoredState struct {
	*LState
	env *LTable
}

func (s *StoredState) snapshot() *StoredState {
	s.env = s.NewTable()
	s.LState.Env.ForEach(func(k LValue, v LValue) {
		s.env.RawSet(k, v)
	})
	return s
}
func (s *StoredState) restore() (r *StoredState) {
	//safeguard
	defer func() {
		rc := recover()
		if rc != nil {
			r = nil
		}
	}()
	s.LState.Pop(s.LState.GetTop())
	s.LState.Env = s.NewTable()
	s.env.ForEach(func(k LValue, v LValue) {
		s.LState.Env.RawSet(k, v)
	})
	return s
}

//StatePool threadsafe LState Pool
type StatePool struct {
	m     sync.Mutex
	saved []*StoredState //TODO replace with more effective structure
	ctor  func() *LState
}

//CreatePoolWith create pool with user defined constructor
//
//**Note** GluModule will auto registered
func CreatePoolWith(ctor func() *LState) *StatePool {
	return &StatePool{saved: make([]*StoredState, 0, PoolSize), ctor: ctor}
}
func CreatePool() *StatePool {
	return &StatePool{saved: make([]*StoredState, 0, PoolSize)}
}

func (pl *StatePool) Get() *StoredState {
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

func (pl *StatePool) new() *StoredState {
	if pl.ctor != nil {
		l := pl.ctor()
		GluModule.PreLoad(l)
		return (&StoredState{LState: l}).snapshot()
	}
	L := NewState(Option)
	configurer(L)
	return (&StoredState{LState: L}).snapshot()
}

func (pl *StatePool) Put(L *StoredState) {
	if L.IsClosed() {
		return
	}
	// reset stack
	l := L.restore()
	if l == nil {
		return
	}
	pl.m.Lock()
	defer pl.m.Unlock()
	pl.saved = append(pl.saved, l)
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
