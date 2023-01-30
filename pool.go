package glu

import (
	"errors"
	. "github.com/yuin/gopher-lua"
	"sync"
)

var (
	//Option LState configuration
	Option      = Options{}
	InitialSize = 4
	pool        *VmPool
)

// MakePool manual create statePool , when need to change Option, should invoke once before use Get and Put
func MakePool() {
	pool = CreatePool()
}

// Get LState from statePool
func Get() *Vm {
	if pool == nil {
		MakePool()
	}
	return pool.Get()
}

// Put LState back to statePool
func Put(s *Vm) {
	if pool == nil {
		MakePool()
	}
	pool.Put(s)
}

var (
	//Auto if true, will autoload modules in registry
	Auto = true
)

// Vm take Env Snapshot to protect from Global pollution
type (
	Vm struct {
		*LState
		env *LTable
		reg *LTable
	}
)

// Polluted check if the Env is polluted
func (s *Vm) Polluted() (r bool) {
	return !s.regEqual() || !s.TabEqualTo(s.env, s.G.Global)
}

// Snapshot take snapshot for Env
func (s *Vm) Snapshot() *Vm {
	s.reg = s.regCopy(s.G.Registry)
	s.env = s.TabCopyNew(s.G.Global)
	return s
}
func (s *Vm) restore() {
	if !s.regEqual() {
		s.G.Registry = s.regCopy(s.reg)
	}
	if !s.TabEqualTo(s.G.Global, s.env) {
		s.G.Global = s.TabCopyNew(s.env)
	}
	s.G.MainThread = nil
	s.G.CurrentThread = nil
	s.Parent = nil
	s.Dead = false
	s.Env = s.G.Global
}

var errNotEqual = errors.New("")

func (s *Vm) regEqual() bool {
	r0 := s.G.Registry
	r1 := s.reg
	return s.TabChildEqualTo(r0, r1, "FILE*", "_LOADED", "_LOADERS")
}
func (s *Vm) regCopy(r *LTable) *LTable {
	return s.TabCopyChildNew(r, "FILE*", "_LOADED", "_LOADERS")
}

func (s *Vm) TabEqualTo(t1 *LTable, t2 *LTable) (r bool) {
	defer func() {
		if e := recover(); e == nil {
			r = true
			return
		} else if e == errNotEqual {
			r = false
			return
		} else {
			panic(e)
		}
	}()
	if t1 == t2 {
		return
	}
	//fast break need test
	t1.ForEach(func(key LValue, val LValue) {
		if t2.RawGet(key) != val {
			panic(errNotEqual)
			return
		}
	})
	return

}
func (s *Vm) TabChildEqualTo(t1 *LTable, t2 *LTable, keys ...string) (r bool) {
	if keys == nil {
		return s.TabEqualTo(t1, t2)
	}
	for _, v := range keys {
		c1 := t1.RawGetString(v)
		c2 := t2.RawGetString(v)
		if c1.Type() != c2.Type() || c1.Type() != LTTable || LTTable != c2.Type() {
			return false
		}
		if !(s.TabEqualTo(c1.(*LTable), c2.(*LTable))) {
			return false
		}
	}
	return true

}
func (s *Vm) TabCopyNew(f *LTable) *LTable {
	t := s.NewTable()
	f.ForEach(t.RawSetH)
	return t
}
func (s *Vm) TabCopyChildNew(f *LTable, keys ...string) *LTable {
	if keys == nil {
		return s.TabCopyNew(f)
	}
	t := s.NewTable()
	for _, v := range keys {
		c := f.RawGetString(v)
		if c.Type() == LTTable {
			t.RawSetString(v, s.TabCopyNew(c.(*LTable)))
		} else if c != LNil {
			t.RawSetString(v, c)
		}

	}
	return t
}

// Reset reset Env
// @fluent
func (s *Vm) Reset() (r *Vm) {
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
		s.restore()
	}
	return s
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
func (s *Vm) OpenLibsWithout(names ...string) *Vm {
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

//region Pool

// VmPool threadsafe LState Pool
type VmPool struct {
	m     sync.Mutex
	saved []*Vm //TODO replace with more effective structure
	ctor  func() *LState
}

// CreatePoolWith create pool with user defined constructor
//
//	GluMod will auto registered
func CreatePoolWith(ctor func() *LState) *VmPool {
	return &VmPool{saved: make([]*Vm, 0, InitialSize), ctor: ctor}
}
func CreatePool() *VmPool {
	return &VmPool{saved: make([]*Vm, 0, InitialSize)}
}

func (pl *VmPool) Get() *Vm {
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

func (pl *VmPool) new() *Vm {
	if pl.ctor != nil {
		l := pl.ctor()
		GluMod.PreLoad(l)
		return (&Vm{LState: l}).Snapshot()
	}
	L := NewState(Option)
	configurer(L)
	return (&Vm{LState: L}).Snapshot()
}

func (pl *VmPool) Put(L *Vm) {
	if L.IsClosed() {
		return
	}
	// reset stack
	l := L.Reset()
	if l == nil {
		return
	}
	pl.m.Lock()
	defer pl.m.Unlock()
	pl.saved = append(pl.saved, l)
}

//Recycle the pool space to max size
func (pl *VmPool) Recycle(max int) {
	if len(pl.saved) > max {
		pl.m.Lock()
		defer pl.m.Unlock()
		pl.saved = pl.saved[:max]
	}
}

func (pl *VmPool) Shutdown() {
	pl.m.Lock()
	defer pl.m.Unlock()
	for _, L := range pl.saved {
		L.Close()
	}
	pl.saved = nil
}

// endregion
func configurer(l *LState) {
	GluMod.PreLoad(l)
	if Auto {
		for _, module := range registry {
			module.PreLoad(l)
		}
	}
}
