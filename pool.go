package glu

import (
	. "github.com/yuin/gopher-lua"
	"sync"
)

var (
	//Option LState configuration
	Option   = Options{}
	PoolSize = 4
	pool     *StatePool
)

// MakePool manual create statePool , when need to change Option, should invoke once before use Get and Put
func MakePool() {
	pool = CreatePool()
}

// Get LState from statePool
func Get() *StoredState {
	if pool == nil {
		MakePool()
	}
	return pool.Get()
}

// Put LState back to statePool
func Put(s *StoredState) {
	if pool == nil {
		MakePool()
	}
	pool.Put(s)
}

var (
	//Auto if true, will autoload modules in registry
	Auto = true
)

//region Pool

// StoredState take Env snapshot to protect from Global pollution
type (
	StoredState struct {
		*LState
		*shadow
	}
	shadow struct {
		env *LTable
		reg *LTable
	}
)

// eqTo check shadow equality
func eqTo(t1 *LTable, t2 *LTable) (r bool) {
	defer func() {
		r = recover() == nil
	}()
	if t1 == t2 {
		return
	}
	//fast break need test
	t1.ForEach(func(key LValue, val LValue) {
		if t2.RawGet(key) != val {
			panic("")
			return
		}
	})
	return
}

// copyTo shadow copy
func copyTo(t1 *LTable, t2 *LTable) {
	if t1 == t2 {
		return
	}
	t1.ForEach(func(key LValue, val LValue) {
		t2.RawSetH(key, val)
	})
}
func snapshot(s *LState) *shadow {
	env := s.NewTable()
	copyTo(s.G.Global, env)
	reg := s.NewTable()
	copyTo(s.G.Registry, reg)
	return &shadow{
		env: env,
		reg: reg,
	}
}
func (a *shadow) equals(s *LState) (r bool) {

	return eqTo(s.G.Global, a.env) && eqTo(s.G.Registry, a.reg)

}
func (a *shadow) reset(s *LState) {
	if !eqTo(s.G.Global, a.env) {
		s.G.Global = s.NewTable()
		copyTo(a.env, s.G.Global)
	}
	if !eqTo(s.G.Registry, a.reg) {
		s.G.Registry = s.NewTable()
		copyTo(a.reg, s.G.Registry)
	}
	s.G.MainThread = nil
	s.G.CurrentThread = nil
	s.Parent = nil
	s.Dead = false
	s.Env = s.G.Global
	//some hack may need
}

// Polluted check if the Env is polluted
func (s *StoredState) Polluted() (r bool) {
	return !s.shadow.equals(s.LState)
}

type lib struct {
	name     string
	register func(state *LState) int
}

var libs = []lib{
	{LoadLibName, OpenPackage},
	{BaseLibName, OpenBase},
	{TabLibName, OpenTable},
	{IoLibName, OpenIo},
	{OsLibName, OpenOs},
	{StringLibName, OpenString},
	{MathLibName, OpenMath},
	{DebugLibName, OpenDebug},
	{ChannelLibName, OpenChannel},
	{CoroutineLibName, OpenCoroutine},
}

// OpenLibsWithout open gopher-lua libs but filter some by name
// @fluent
func (s *StoredState) OpenLibsWithout(names ...string) *StoredState {
	tb := s.FindTable(s.Get(RegistryIndex).(*LTable), "_LOADED", 1)
	for _, b := range libs {
		name := b.name
		exclude := false
		for _, n := range names {
			if n == name {
				exclude = true
				break
			}
		}
		mod := s.GetField(tb, name)
		//clean if loaded
		if mod.Type() == LTTable && exclude {
			s.SetField(tb, name, LNil)
		} else if mod.Type() != LTTable && !exclude {
			s.Push(s.NewFunction(b.register))
			s.Push(LString(name))
			s.Call(1, 0)
		}
	}
	return s
}

// snapshot take snapshot for Env
func (s *StoredState) snapshot() *StoredState {
	s.shadow = snapshot(s.LState)
	return s
}

// restore reset Env
// @fluent
func (s *StoredState) restore() (r *StoredState) {
	//safeguard
	defer func() {
		rc := recover()
		if rc != nil {
			r = nil
		}
	}()
	s.LState.Pop(s.LState.GetTop())
	if s.Polluted() {
		// reset global https://github.com/ZenLiuCN/glu/issues/1
		s.shadow.reset(s.LState)
	}
	return s
}

// StatePool threadsafe LState Pool
type StatePool struct {
	m     sync.Mutex
	saved []*StoredState //TODO replace with more effective structure
	ctor  func() *LState
}

// CreatePoolWith create pool with user defined constructor
//
//	GluModule will auto registered
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

// endregion
func configurer(l *LState) {
	GluModule.PreLoad(l)
	if Auto {
		for _, module := range registry {
			module.PreLoad(l)
		}
	}
}
