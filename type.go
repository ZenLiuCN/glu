package glu

import (
	"fmt"
	. "github.com/yuin/gopher-lua"
	"strings"
)

// noinspection GoSnakeCaseUsage,GoUnusedConst
const (
	OPERATE_INVALID  Operate = iota
	OPERATE_ADD              // +
	OPERATE_SUB              // -
	OPERATE_MUL              // *
	OPERATE_DIV              // /
	OPERATE_UNM              // -
	OPERATE_MOD              // %
	OPERATE_POW              // ^
	OPERATE_CONCAT           // ..
	OPERATE_EQ               // ==
	OPERATE_LT               // <
	OPERATE_LE               // <=
	OPERATE_LEN              // #
	OPERATE_INDEX            // []
	OPERATE_NEWINDEX         // []=
	OPERATE_TOSTRING         // tostring
	OPERATE_CALL             // ()
)

type Operate int

type Type[T any] interface {
	Modular

	// New create new instance and push on stack
	New(l *LState, val T) int
	// NewValue create new LValue
	NewValue(l *LState, val T) *LUserData
	// Cast check the type can use cast (when construct with NewTypeCast)
	Cast() bool
	// Check  cast value on stack (already have error processed)
	Check(s *LState, n int) T
	// CheckSelf  receiver on stack (already have error processed)
	CheckSelf(s *LState) T
	// CheckUserData cast UserData (already have error processed)
	CheckUserData(ud *LUserData, s *LState) T
	// Caster  cast value
	Caster() func(any) (T, bool)
	// AddFunc static function
	AddFunc(name string, help string, fn LGFunction) Type[T]

	// AddField static field
	AddField(name string, help string, value LValue) Type[T]
	// AddFieldSupplier static field with supplier
	AddFieldSupplier(name string, help string, su func(s *LState) LValue) Type[T]
	//AddModule add sub-module
	AddModule(mod Modular) Type[T]
	// AddMethod add method to this type which means instance method.
	AddMethod(name string, help string, value LGFunction) Type[T]

	// AddMethodUserData add method to this type which means instance method, with auto extract first argument.
	AddMethodUserData(name string, help string, act func(s *LState, u *LUserData) int) Type[T]

	// AddMethodCast prechecked type (only create with NewTypeCast).
	AddMethodCast(name string, help string, act func(s *LState, i T) int) Type[T]

	// Override operators an operator
	Override(op Operate, help string, fn LGFunction) Type[T]

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
	constructor func(*LState) T //Constructor for this BaseType , also can define other Constructor by add functions
	methods     map[string]funcInfo
	fields      map[string]fieldInfo
	operators   map[Operate]funcInfo
}

// NewSimpleType create new BaseType without ctor
func NewSimpleType[T any](name string, help string, top bool) *BaseType[T] {
	return &BaseType[T]{Mod: &Mod{Name: name, Top: top, Help: help}}
}

// NewType create new BaseType
func NewType[T any](name string, help string, top bool, ctorHelp string, ctor func(*LState) (v T)) *BaseType[T] {
	return &BaseType[T]{Mod: &Mod{Name: name, Top: top, Help: help}, constructor: ctor, HelpCtor: ctorHelp}
}

// NewTypeCast create new BaseType with reflect Signature
func NewTypeCast[T any](caster func(a any) (v T, ok bool), name string, help string, top bool, ctorHelp string, ctor func(s *LState) (v T)) *BaseType[T] {
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
func (m *BaseType[T]) AddFieldSupplier(name string, help string, su func(s *LState) LValue) Type[T] {
	m.Mod.AddFieldSupplier(name, help, su)
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
			if value.Supplier != nil {
				l.SetField(mt, key, value.Supplier(l))
			} else if value.Value != LNil {
				l.SetField(mt, key, value.Value)
			} else {
				panic(fmt.Errorf(`invalid field info`))
			}
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
			case OPERATE_TOSTRING:
				name = "__tostring"
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
	val := m.constructor(s)
	ud := s.NewUserData()
	ud.Value = val
	s.SetMetatable(ud, s.GetTypeMetatable(m.Mod.Name))
	s.Push(ud)
	return 1
}

// Cast check the type can use cast (when construct with NewTypeCast)
func (m BaseType[T]) Cast() bool {
	return m.caster != nil
}

// Check  cast value on stack
func (m BaseType[T]) Check(s *LState, n int) T {
	v, ok := m.caster(s.CheckUserData(n).Value)
	if !ok {
		s.ArgError(n, "require type "+m.Mod.Name)
	}
	return v

}
func (m BaseType[T]) CheckUserData(ud *LUserData, s *LState) T {
	v, ok := m.caster(ud.Value)
	if !ok {
		s.ArgError(1, "require receiver type "+m.Mod.Name)
	}
	return v
}
func (m BaseType[T]) CheckSelf(s *LState) T {
	v, ok := m.caster(s.CheckUserData(1).Value)
	if !ok {
		s.ArgError(1, "require receiver type "+m.Mod.Name)
	}
	return v
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

// AddMethodUserData add method to this type which means instance method, with auto extract first argument.
func (m *BaseType[T]) AddMethodUserData(name string, help string, act func(s *LState, data *LUserData) int) Type[T] {
	return m.AddMethod(name, help, func(s *LState) int {
		u := s.CheckUserData(1)
		if u == nil {
			return 0
		}
		return act(s, u)
	})
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

// AddMethodCast prechecked type (only create with NewTypeCast).
func (m *BaseType[T]) AddMethodCast(name string, help string, act func(s *LState, data T) int) Type[T] {
	if !m.Cast() {
		panic("can't use AddMethodCast for not create with NewTypeCast")
	}
	return m.AddMethod(name, help, func(s *LState) int {
		return act(s, m.CheckSelf(s))
	})
}

// OverrideCast see Override and AddMethodCast
func (m *BaseType[T]) OverrideCast(op Operate, help string, act func(s *LState, data T) int) Type[T] {
	if !m.Cast() {
		panic("can't use OverrideCast for not create with NewTypeCast")
	}
	return m.Override(op, help, func(s *LState) int {
		return act(s, m.CheckSelf(s))
	})
}
