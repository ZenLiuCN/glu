package glu

import (
	"fmt"
	. "github.com/yuin/gopher-lua"
	"reflect"
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
type Type interface {
	Modular
	// Type the real go type
	Type() reflect.Type
	// New create new instance
	New(l *LState, val interface{}) int
	// NewValue create new LValue
	NewValue(l *LState, val interface{}) *LUserData
	// CanCast check the type can use cast (when construct with NewTypeCast)
	CanCast() bool
	// CastVar  cast value on stack (already have error processed)
	CastVar(s *LState, n int) interface{}
	// Cast nil if not current type else the value
	Cast(u *LUserData) interface{}

	// AddFunc static function
	AddFunc(name string, help string, fn LGFunction) Type
	// AddField static field
	AddField(name string, help string, value LValue) Type

	// AddMethod add method to this type which means instance method.
	AddMethod(name string, help string, value LGFunction) Type

	// AddMethodUserData add method to this type which means instance method, with auto extract first argument.
	AddMethodUserData(name string, help string, act func(s *LState, u *LUserData) int) Type

	// AddMethodCast prechecked type (only create with NewTypeCast).
	AddMethodCast(name string, help string, act func(s *LState, i interface{}) int) Type

	// Override override an operator
	Override(op Operate, help string, fn LGFunction) Type

	// OverrideUserData see Override and AddMethodUserData
	OverrideUserData(op Operate, help string, act func(s *LState, u *LUserData) int) Type

	// OverrideCast see Override and AddMethodCast
	OverrideCast(op Operate, help string, act func(s *LState, i interface{}) int) Type
}

// BaseType define a LTable with MetaTable, which mimicry class like action in Lua
type BaseType struct {
	Mod
	HelpCtor    string
	signature   reflect.Type              //reflect.Type
	constructor func(*LState) interface{} //Constructor for this BaseType , also can define other Constructor by add functions
	methods     map[string]funcInfo
	fields      map[string]fieldInfo
	override    map[Operate]funcInfo
}

// NewSimpleType create new BaseType without ctor
func NewSimpleType(name string, help string, top bool) *BaseType {
	return &BaseType{Mod: Mod{Name: name, Top: top, Help: help}}
}

// NewType create new BaseType
func NewType(name string, help string, top bool, ctorHelp string, ctor func(*LState) interface{}) *BaseType {
	return &BaseType{Mod: Mod{Name: name, Top: top, Help: help}, constructor: ctor, HelpCtor: ctorHelp}
}

// NewTypeCast create new BaseType with reflect Signature
func NewTypeCast(sample interface{}, name string, help string, top bool, ctorHelp string, ctor func(*LState) interface{}) *BaseType {
	return &BaseType{Mod: Mod{Name: name, Top: top, Help: help}, signature: reflect.TypeOf(sample), constructor: ctor, HelpCtor: ctorHelp}
}
func (m *BaseType) Type() reflect.Type {
	return m.signature
}
func (m *BaseType) prepare() {
	if m.prepared {
		return
	}
	help := make(map[string]string)
	mh := new(strings.Builder) //mod help builder
	if m.Help != "" {
		mh.WriteString(m.Help)
		mh.WriteRune('\n')
	} else {
		mh.WriteString(m.Name)
		mh.WriteRune('\n')
	}

	if len(m.functions) > 0 {
		for s, info := range m.functions {
			if info.Help != "" {
				help[s] = info.Help
				mh.WriteString(fmt.Sprintf("%s.%s %s\n", m.Name, s, info.Help))
			} else {
				mh.WriteString(fmt.Sprintf("%s.%s\n", m.Name, s))
			}
		}
	}
	if len(m.fields) > 0 {
		for s, value := range m.fields {
			if value.Help != "" {
				help[s] = value.Help
				mh.WriteString(fmt.Sprintf("%s.%s %s\n", m.Name, s, value.Help))
			} else {
				mh.WriteString(fmt.Sprintf("%s.%s\n", m.Name, s))
			}
		}
	}
	if len(m.methods) > 0 {
		for s, value := range m.methods {
			if value.Help != "" {
				help[s] = value.Help
				mh.WriteString(fmt.Sprintf("%s.%s %s\n", m.Name, s, value.Help))
			} else {
				mh.WriteString(fmt.Sprintf("%s.%s\n", m.Name, s))
			}
		}
	}
	if len(m.submodules) > 0 {
		for _, t := range m.submodules {
			mh.WriteString(fmt.Sprintf("%s.%s \n", m.Name, t.GetName()))
		}
	}

	if m.constructor != nil {
		help["new"] = m.HelpCtor
		mh.WriteString(fmt.Sprintf("%s.new %s\n", m.Name, m.HelpCtor))
	}
	if len(m.methods) > 0 {
		for s, info := range m.methods {
			if info.Help != "" {
				help[s] = info.Help
				mh.WriteString(fmt.Sprintf("%s::%s %s\n", m.Name, s, info.Help))
			} else {
				mh.WriteString(fmt.Sprintf("%s::%s\n", m.Name, s))
			}
		}
	}
	if len(m.override) > 0 {
		for op, info := range m.override {
			var sym string
			var name string
			switch op {
			case OPERATE_ADD:
				name = "__add"
				sym = "+"
			case OPERATE_SUB:
				name = "__sub"
				sym = "-"
			case OPERATE_MUL:
				name = "__mul"
				sym = "*"
			case OPERATE_DIV:
				name = "__div"
				sym = "/"
			case OPERATE_UNM:
				name = "__unm"
				sym = "-"
			case OPERATE_MOD:
				name = "__mod"
				sym = "%"
			case OPERATE_POW:
				name = "__pow"
				sym = "^"
			case OPERATE_CONCAT:
				name = "__concat"
				sym = ".."
			case OPERATE_EQ:
				name = "__eq"
				sym = "=="
			case OPERATE_LT:
				name = "__lt"
				sym = "<"
			case OPERATE_LE:
				name = "__le"
				sym = "<="
			case OPERATE_LEN:
				name = "__len"
				sym = "#"
			case OPERATE_NEWINDEX:
				name = "__newindex"
				sym = "[]="
			case OPERATE_TO_STRING:
				name = "__to_string"
				sym = "tostring"
			case OPERATE_CALL:
				name = "__call"
				sym = "()"
			case OPERATE_INDEX:
				if len(m.methods) > 0 {
					panic(ErrIndexOverrideWithMethods)
				}
				name = "__index"
				sym = "[]"
			default:
				panic(fmt.Errorf("unsupported override of %d", op))
			}
			if info.Help != "" {
				help[name] = info.Help
				mh.WriteString(fmt.Sprintf("%s::%s %s\n", m.Name, sym, info.Help))
			} else {
				mh.WriteString(fmt.Sprintf("%s::%s\n", m.Name, sym))
			}
		}

	}

	if mh.Len() > 0 {
		help[HelpKey] = mh.String()
	}
	m.prepared = true

}

// AddFunc add function to this Modular
//
// @name function name, must match lua limitation
//
// @help help string, if empty will not generate into help
//
// @fn the LGFunction
func (m *BaseType) AddFunc(name string, help string, fn LGFunction) Type {
	m.Mod.AddFunc(name, help, fn)
	return m

}

//SafeFun warp with SafeFunc
func (m *BaseType) SafeFun(name string, help string, fn LGFunction) Type {
	m.Mod.AddFunc(name, help, SafeFunc(fn))
	return m

}

// AddField add value field to this Modular
//
// @name the field name
//
// @help help string, if empty will not generate into help
//
// @value the field value
func (m *BaseType) AddField(name string, help string, value LValue) Type {
	m.Mod.AddField(name, help, value)
	return m
}

// AddModule add sub-module to this Modular
//
// @mod the Mod **Note** must with TopLevel false.
func (m *BaseType) AddModule(mod Modular) Type {
	m.Mod.AddModule(mod)
	return m

}
func (m *BaseType) PreLoad(l *LState) {
	if !m.Top {
		return
	}

	l.SetGlobal(m.Name, m.getOrBuildMeta(l))
}
func (m *BaseType) PreloadSubModule(l *LState, t *LTable) {
	if m.Top {
		return
	}
	t.RawSetString(m.Name, m.getOrBuildMeta(l))
}

func (m *BaseType) getOrBuildMeta(l *LState) *LTable {
	m.prepare()
	var mt *LTable
	var ok bool
	if mt, ok = l.GetTypeMetatable(m.Name).(*LTable); ok {
		return mt
	}
	mt = l.NewTypeMetatable(m.Name)
	fn := make(map[string]LGFunction)
	if m.constructor != nil {
		l.SetField(mt, "new", l.NewFunction(m.new))
	}
	if len(m.functions) > 0 {
		for s, info := range m.functions {
			fn[s] = info.Func
		}
	}
	if len(m.fields) > 0 {
		for key, value := range m.fields {
			l.SetField(mt, key, value.Value)
		}
	}
	if len(m.submodules) > 0 {
		for _, t := range m.submodules {
			t.PreloadSubModule(l, mt)
		}
	}
	if len(m.methods) > 0 {
		method := make(map[string]LGFunction, len(m.functions))
		for s, info := range m.methods {
			method[s] = info.Func
		}
		// methods
		mt.RawSetString("__index", l.SetFuncs(l.NewTable(), method))
		//l.SetField(mt, "__index", l.SetFuncs(l.NewTable(), method))
	}
	if len(m.override) > 0 {
		for op, info := range m.override {
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
				panic(fmt.Errorf("unsupported override of %d", op))
			}
			l.SetField(mt, name, l.NewFunction(info.Func))
		}

	}
	if len(m.help) > 0 {
		fn[HelpFunc] = helpFn(m.help)
	}
	if len(fn) > 0 {
		l.SetFuncs(mt, fn)
	}
	return mt
}

// New wrap an instance into LState
func (m BaseType) New(l *LState, val interface{}) int {
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, m.getOrBuildMeta(l))
	l.Push(ud)
	return 1
}

// NewValue create new LValue
func (m BaseType) NewValue(l *LState, val interface{}) *LUserData {
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, m.getOrBuildMeta(l))
	return ud
}

// new internal creator
func (m BaseType) new(l *LState) int {
	val := m.constructor(l)
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, l.GetTypeMetatable(m.Name))
	l.Push(ud)
	return 1
}

// CanCast check the type can use cast (when construct with NewTypeCast)
func (m BaseType) CanCast() bool {
	return m.signature != nil
}

// CastVar  cast value on stack
func (m BaseType) CastVar(s *LState, n int) interface{} {
	u := s.CheckUserData(n)
	if u == nil {
		return nil
	}
	v := m.Cast(u)
	if v == nil {
		s.ArgError(n, "require type "+m.Name)
		return nil
	}
	return v

}

// Cast nil if not current type else the value
func (m BaseType) Cast(u *LUserData) interface{} {
	if u == nil || reflect.TypeOf(u.Value) != m.signature {
		return nil
	}
	return u.Value
}

// AddMethod add method to this type which means instance method.
func (m *BaseType) AddMethod(name string, help string, value LGFunction) Type {
	if m.methods == nil {
		m.methods = make(map[string]funcInfo)
	} else if _, ok := m.methods[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.methods[name] = funcInfo{help, value}
	return m
}

//SafeMethod warp with SafeFunc
func (m *BaseType) SafeMethod(name string, help string, value LGFunction) Type {
	if m.methods == nil {
		m.methods = make(map[string]funcInfo)
	} else if _, ok := m.methods[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.methods[name] = funcInfo{help, SafeFunc(value)}
	return m
}

// AddMethodUserData add method to this type which means instance method, with auto extract first argument.
func (m *BaseType) AddMethodUserData(name string, help string, act func(s *LState, data *LUserData) int) Type {
	return m.AddMethod(name, help, func(s *LState) int {
		u := s.CheckUserData(1)
		if u == nil {
			return 0
		}
		return act(s, u)
	})
}

// AddMethodCast prechecked type (only create with NewTypeCast).
func (m *BaseType) AddMethodCast(name string, help string, act func(s *LState, data interface{}) int) Type {
	if !m.CanCast() {
		panic("can't use AddMethodCast for not create with NewTypeCast")
	}
	return m.AddMethod(name, help, func(s *LState) int {
		u := s.CheckUserData(1)
		if u == nil {
			return 0
		}
		d := m.Cast(u)
		if d == nil {
			return 0
		}
		return act(s, d)
	})
}

// Override override an operator
func (m *BaseType) Override(op Operate, help string, fn LGFunction) Type {
	if m.override == nil {
		m.override = make(map[Operate]funcInfo)
	} else if _, ok := m.override[op]; ok {
		panic(ErrAlreadyExists)
	}
	m.override[op] = funcInfo{help, fn}
	return m
}

//SafeOverride wrap with SafeFunc
func (m *BaseType) SafeOverride(op Operate, help string, fn LGFunction) Type {
	if m.override == nil {
		m.override = make(map[Operate]funcInfo)
	} else if _, ok := m.override[op]; ok {
		panic(ErrAlreadyExists)
	}
	m.override[op] = funcInfo{help, SafeFunc(fn)}
	return m
}

// OverrideUserData see Override and AddMethodUserData
func (m *BaseType) OverrideUserData(op Operate, help string, act func(s *LState, data *LUserData) int) Type {
	return m.Override(op, help, func(s *LState) int {
		u := s.CheckUserData(1)
		if u == nil {
			return 0
		}
		return act(s, u)
	})
}

// OverrideCast see Override and AddMethodCast
func (m *BaseType) OverrideCast(op Operate, help string, act func(s *LState, data interface{}) int) Type {
	if !m.CanCast() {
		panic("can't use OverrideCast for not create with NewTypeCast")
	}
	return m.Override(op, help, func(s *LState) int {
		d := m.CastVar(s, 1)
		if d == nil {
			return 0
		}
		return act(s, d)
	})
}
