package glu

import (
	"errors"
	"fmt"
	. "github.com/yuin/gopher-lua"
	"strings"
)

// noinspection GoSnakeCaseUsage,GoUnusedConst
const (
	OPERATE_INVALID   Operate = iota
	OPERATE_ADD               // +
	OPERATE_SUB               // -
	OPERATE_MUL               // *
	OPERATE_DIV               // /
	OPERATE_UNM               // -
	OPERATE_MOD               // %
	OPERATE_POW               // ^
	OPERATE_CONCAT            // ..
	OPERATE_EQ                // ==
	OPERATE_LT                // <
	OPERATE_LE                // <=
	OPERATE_LEN               // #
	OPERATE_INDEX             // []
	OPERATE_NEWINDEX          // []=
	OPERATE_TO_STRING         // tostring
	OPERATE_CALL              // ()
)

type Operate int

type Type[T any] interface {
	Modular

	// New create new instance and push on stack
	New(l *LState, val T) int
	// NewValue create new LValue
	NewValue(l *LState, val T) *LUserData
	// CanCast check the type can use cast (when construct with NewTypeCast)
	CanCast() bool
	// CastVar  cast value on stack (already have error processed)
	CastVar(s *LState, n int) (T, bool)
	// Cast  receiver on stack (already have error processed)
	Cast(s *LState) (T, bool)
	// CastUserData cast UserData (already have error processed)
	CastUserData(ud *LUserData, s *LState) (T, bool)
	// Caster  cast value
	Caster() func(any) (T, bool)

	// AddFunc static function
	AddFunc(name string, help string, fn LGFunction) Type[T]
	//SafeFun warp with SafeFunc
	SafeFun(name string, help string, value LGFunction) Type[T]

	// AddField static field
	AddField(name string, help string, value LValue) Type[T]
	//SafeMethod warp with SafeFunc
	SafeMethod(name string, help string, value LGFunction) Type[T]
	// AddMethod add method to this type which means instance method.
	AddMethod(name string, help string, value LGFunction) Type[T]

	// AddMethodUserData add method to this type which means instance method, with auto extract first argument.
	AddMethodUserData(name string, help string, act func(s *LState, u *LUserData) int) Type[T]

	// AddMethodCast prechecked type (only create with NewTypeCast).
	AddMethodCast(name string, help string, act func(s *LState, i T) int) Type[T]

	// Override operators an operator
	Override(op Operate, help string, fn LGFunction) Type[T]

	//SafeOverride warp with SafeFunc
	SafeOverride(op Operate, help string, value LGFunction) Type[T]

	// OverrideUserData see Override and AddMethodUserData
	OverrideUserData(op Operate, help string, act func(s *LState, u *LUserData) int) Type[T]

	// OverrideCast see Override and AddMethodCast
	OverrideCast(op Operate, help string, act func(s *LState, i T) int) Type[T]
}

// BaseType define a LTable with MetaTable, which mimicry class like action in Lua
type BaseType[T any] struct {
	Mod         *Mod
	HelpCtor    string
	caster      func(any) (T, bool)
	defVal      T
	constructor func(*LState) (T, bool) //Constructor for this BaseType , also can define other Constructor by add functions
	methods     map[string]funcInfo
	fields      map[string]fieldInfo
	operators   map[Operate]funcInfo
}

// NewSimpleType create new BaseType without ctor
func NewSimpleType[T any](name string, help string, top bool) *BaseType[T] {
	return &BaseType[T]{Mod: &Mod{Name: name, Top: top, Help: help}}
}

// NewType create new BaseType
func NewType[T any](name string, help string, top bool, ctorHelp string, ctor func(*LState) (T, bool)) *BaseType[T] {
	return &BaseType[T]{Mod: &Mod{Name: name, Top: top, Help: help}, constructor: ctor, HelpCtor: ctorHelp}
}

// NewTypeCast create new BaseType with reflect Signature
func NewTypeCast[T any](caster func(a any) (v T, ok bool), name string, help string, top bool, ctorHelp string, ctor func(s *LState) (v T, ok bool)) *BaseType[T] {
	return &BaseType[T]{Mod: &Mod{Name: name, Top: top, Help: help}, caster: caster, constructor: ctor, HelpCtor: ctorHelp}
}

func (m *BaseType[T]) TopLevel() bool {
	return m.Mod.TopLevel()
}

func (m *BaseType[T]) GetName() string {
	return m.Mod.GetName()
}

func (m *BaseType[T]) GetHelp() string {
	return m.Mod.GetHelp()
}

func (m *BaseType[T]) prepare() {
	if m.Mod.prepared {
		return
	}
	help := make(map[string]string)
	mh := new(strings.Builder) //mod HelpCache builder
	if m.Mod.Help != "" {
		mh.WriteString(m.Mod.Help)
		mh.WriteRune('\n')
	} else {
		mh.WriteString(m.Mod.Name)
		mh.WriteRune('\n')
	}
	helpFuncReg(m.Mod.functions, help, mh, m.Mod.Name)
	helpFieldReg(m.fields, help, mh, m.Mod.Name)
	helpMethodReg(m.methods, help, mh, m.Mod.Name)
	helpOperatorReg(m.operators, len(m.methods) > 0, help, mh, m.Mod.Name)
	helpSubModReg(m.Mod.Submodules, help, mh, m.Mod.Name)
	helpCtorReg(m.constructor, m.HelpCtor, help, mh, m.Mod.Name)
	if mh.Len() > 0 {
		help[HelpKey] = mh.String()
	}
	//prepare sub modules?
	if EagerHelpPrepare && len(m.Mod.Submodules) > 0 {
		for _, sub := range m.Mod.Submodules {
			switch sub.(type) {
			case Prepare:
				sub.(Prepare).prepare()
			default:
				panic("unknown sub type")
			}
		}
	}
	m.Mod.HelpCache = help
	m.Mod.prepared = true

}

// AddFunc add function to this Modular
//
// @name function name, must match lua limitation
//
// @HelpCache HelpCache string, if empty will not generate into HelpCache
//
// @fn the LGFunction
func (m *BaseType[T]) AddFunc(name string, help string, fn LGFunction) Type[T] {
	m.Mod.AddFunc(name, help, fn)
	return m

}

//SafeFun warp with SafeFunc
func (m *BaseType[T]) SafeFun(name string, help string, fn LGFunction) Type[T] {
	m.Mod.AddFunc(name, help, SafeFunc(fn))
	return m

}

// AddField add value field to this Modular
//
// @name the field name
//
// @HelpCache HelpCache string, if empty will not generate into HelpCache
//
// @value the field value
func (m *BaseType[T]) AddField(name string, help string, value LValue) Type[T] {
	m.Mod.AddField(name, help, value)
	return m
}

// AddModule add sub-module to this Modular
//
// @mod the Mod **Note** must with TopLevel false.
func (m *BaseType[T]) AddModule(mod Modular) Type[T] {
	m.Mod.AddModule(mod)
	return m

}
func (m *BaseType[T]) PreLoad(l *LState) {
	if !m.Mod.Top {
		return
	}

	l.SetGlobal(m.Mod.Name, m.getOrBuildMeta(l))
}
func (m *BaseType[T]) PreloadSubModule(l *LState, t *LTable) {
	if m.Mod.Top {
		return
	}
	t.RawSetString(m.Mod.Name, m.getOrBuildMeta(l))
}

func (m *BaseType[T]) getOrBuildMeta(l *LState) *LTable {
	m.prepare()
	var mt *LTable
	var ok bool
	if mt, ok = l.GetTypeMetatable(m.Mod.Name).(*LTable); ok {
		return mt
	}
	mt = l.NewTypeMetatable(m.Mod.Name)
	fn := make(map[string]LGFunction)
	if m.constructor != nil {
		l.SetField(mt, "new", l.NewFunction(m.new))
	}
	if len(m.Mod.functions) > 0 {
		for s, info := range m.Mod.functions {
			fn[s] = info.Func
		}
	}
	if len(m.fields) > 0 {
		for key, value := range m.fields {
			l.SetField(mt, key, value.Value)
		}
	}
	if len(m.Mod.Submodules) > 0 {
		for _, t := range m.Mod.Submodules {
			t.PreloadSubModule(l, mt)
		}
	}
	if len(m.methods) > 0 {
		method := make(map[string]LGFunction, len(m.Mod.functions))
		for s, info := range m.methods {
			method[s] = info.Func
		}
		// methods
		mt.RawSetString("__index", l.SetFuncs(l.NewTable(), method))
		//l.SetField(mt, "__index", l.SetFuncs(l.NewTable(), method))
	}
	if len(m.operators) > 0 {
		for op, info := range m.operators {
			var name string
			switch op {
			case OPERATE_ADD:
				name = "__add"
			case OPERATE_SUB:
				name = "__sub"
			case OPERATE_MUL:
				name = "__mul"
			case OPERATE_DIV:
				name = "__div"
			case OPERATE_UNM:
				name = "__unm"
			case OPERATE_MOD:
				name = "__mod"
			case OPERATE_POW:
				name = "__pow"
			case OPERATE_CONCAT:
				name = "__concat"
			case OPERATE_EQ:
				name = "__eq"
			case OPERATE_LT:
				name = "__lt"
			case OPERATE_LE:
				name = "__le"
			case OPERATE_LEN:
				name = "__len"
			case OPERATE_NEWINDEX:
				name = "__newindex"
			case OPERATE_TO_STRING:
				name = "__to_string"
			case OPERATE_CALL:
				name = "__call"
			case OPERATE_INDEX:
				if len(m.methods) > 0 {
					panic(ErrIndexOverrideWithMethods)
				}
				name = "__index"
			default:
				panic(fmt.Errorf("unsupported operators of %d", op))
			}
			l.SetField(mt, name, l.NewFunction(info.Func))
		}

	}
	if len(m.Mod.HelpCache) > 0 {
		fn[HelpFunc] = helpFn(m.Mod.HelpCache)
	}
	if len(fn) > 0 {
		l.SetFuncs(mt, fn)
	}
	return mt
}

// New wrap an instance into LState
func (m BaseType[T]) New(l *LState, val T) int {
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, m.getOrBuildMeta(l))
	l.Push(ud)
	return 1
}

// NewValue create new LValue
func (m BaseType[T]) NewValue(l *LState, val T) *LUserData {
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, m.getOrBuildMeta(l))
	return ud
}

// new internal creator
func (m BaseType[T]) new(s *LState) (n int) {
	defer func() {
		if r := recover(); r != nil {

			switch r.(type) {
			case error:
				if errors.Is(r.(error), ErrorSuppress) {
					break
				}
				s.RaiseError("error:%s", r.(error).Error())
			case string:
				s.RaiseError("error:%s", r.(string))
			default:
				s.RaiseError("error:%#v", r)
			}
			n = 0
		}
	}()
	val, ok := m.constructor(s)
	if !ok {
		return 0
	}
	ud := s.NewUserData()
	ud.Value = val
	s.SetMetatable(ud, s.GetTypeMetatable(m.Mod.Name))
	s.Push(ud)
	return 1
}

// CanCast check the type can use cast (when construct with NewTypeCast)
func (m BaseType[T]) CanCast() bool {
	return m.caster != nil
}

// CastVar  cast value on stack
func (m BaseType[T]) CastVar(s *LState, n int) (T, bool) {
	u := s.CheckUserData(n)
	if u == nil {
		return m.defVal, false
	}
	v, ok := m.caster(u.Value)
	if !ok {
		s.ArgError(n, "require type "+m.Mod.Name)
		return m.defVal, false
	}
	return v, true

}
func (m BaseType[T]) CastUserData(ud *LUserData, s *LState) (T, bool) {
	v, ok := m.caster(ud.Value)
	if !ok {
		s.ArgError(1, "require receiver type "+m.Mod.Name)
		return m.defVal, false
	}
	return v, true
}
func (m BaseType[T]) Cast(s *LState) (T, bool) {
	u := s.CheckUserData(1)
	if u == nil {
		return m.defVal, false
	}
	v, ok := m.caster(u.Value)
	if !ok {
		s.ArgError(1, "require receiver type "+m.Mod.Name)
		return m.defVal, false
	}
	return v, true
}
func (m *BaseType[T]) Caster() func(any) (T, bool) {
	return m.caster
}

// AddMethod add method to this type which means instance method.
func (m *BaseType[T]) AddMethod(name string, help string, value LGFunction) Type[T] {
	if m.methods == nil {
		m.methods = make(map[string]funcInfo)
	} else if _, ok := m.methods[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.methods[name] = funcInfo{help, value}
	return m
}

//SafeMethod warp with SafeFunc
func (m *BaseType[T]) SafeMethod(name string, help string, value LGFunction) Type[T] {
	if m.methods == nil {
		m.methods = make(map[string]funcInfo)
	} else if _, ok := m.methods[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.methods[name] = funcInfo{help, SafeFunc(value)}
	return m
}

// AddMethodUserData add method to this type which means instance method, with auto extract first argument.
func (m *BaseType[T]) AddMethodUserData(name string, help string, act func(s *LState, data *LUserData) int) Type[T] {
	return m.AddMethod(name, help, func(s *LState) int {
		defer func() {
			if r := recover(); r == nil {
				s.Pop(1)
			} else {
				panic(r)
			}
		}()
		u := s.CheckUserData(1)
		if u == nil {
			return 0
		}
		return act(s, u)
	})
}

// AddMethodCast prechecked type (only create with NewTypeCast).
func (m *BaseType[T]) AddMethodCast(name string, help string, act func(s *LState, data T) int) Type[T] {
	if !m.CanCast() {
		panic("can't use AddMethodCast for not create with NewTypeCast")
	}
	return m.AddMethod(name, help, func(s *LState) int {
		d, ok := m.Cast(s)
		if !ok {
			return 0
		}
		return act(s, d)
	})
}

// Override operators an operator
func (m *BaseType[T]) Override(op Operate, help string, fn LGFunction) Type[T] {
	if m.operators == nil {
		m.operators = make(map[Operate]funcInfo)
	} else if _, ok := m.operators[op]; ok {
		panic(ErrAlreadyExists)
	}
	m.operators[op] = funcInfo{help, fn}
	return m
}

//SafeOverride wrap with SafeFunc
func (m *BaseType[T]) SafeOverride(op Operate, help string, fn LGFunction) Type[T] {
	if m.operators == nil {
		m.operators = make(map[Operate]funcInfo)
	} else if _, ok := m.operators[op]; ok {
		panic(ErrAlreadyExists)
	}
	m.operators[op] = funcInfo{help, SafeFunc(fn)}
	return m
}

// OverrideUserData see Override and AddMethodUserData
func (m *BaseType[T]) OverrideUserData(op Operate, help string, act func(s *LState, data *LUserData) int) Type[T] {
	return m.Override(op, help, func(s *LState) int {
		u := s.CheckUserData(1)
		if u == nil {
			return 0
		}
		return act(s, u)
	})
}

// OverrideCast see Override and AddMethodCast
func (m *BaseType[T]) OverrideCast(op Operate, help string, act func(s *LState, data T) int) Type[T] {
	if !m.CanCast() {
		panic("can't use OverrideCast for not create with NewTypeCast")
	}
	return m.Override(op, help, func(s *LState) int {
		d, ok := m.CastVar(s, 1)
		if !ok {
			return 0
		}
		return act(s, d)
	})
}
