// Package glu  support yuin/gopher-lua with easy modular definition and other enchantments.
// glu.Modular and gua.Type will inject mod.Help(name string?) method to output help information.
// glu.Get: Pool function to get a lua.LState.
// glu.Put: Pool function to return a lua.LState.
// glu.registry: shared module registry.
// glu.Auto: config for autoload modules in registry into lua.LState.
package glu

import (
	"errors"
	"fmt"
	. "github.com/yuin/gopher-lua"
	"reflect"
	"strings"
)

var (
	registry   []Modular
	names      map[string]struct{}
	EXIST_NODE = struct{}{}
)

// Register modular into registry
func Register(m ...Modular) (err error) {
	for _, mod := range m {
		if v, ok := names[mod.GetName()]; ok && v == EXIST_NODE {
			return ErrAlreadyExists
		}
		registry = append(registry, mod)
		names[mod.GetName()] = EXIST_NODE
	}
	return
}

var (
	ErrAlreadyExists            = errors.New("element already exists")
	ErrIndexOverrideWithMethods = errors.New("element both have methods and index overrides")
	ErrIsTop                    = errors.New("element is top module")
)

const (
	OPERATE_NONE Operate = iota
	OPERATE_ADD
	OPERATE_SUB
	OPERATE_MUL
	OPERATE_DIV
	OPERATE_UNM
	OPERATE_MOD
	OPERATE_POW
	OPERATE_CONCAT
	OPERATE_EQ
	OPERATE_LT
	OPERATE_LE
	OPERATE_LEN
	OPERATE_INDEX
	OPERATE_NEWINDEX
	OPERATE_TO_STRING
	OPERATE_CALL
)

type (
	Operate int
	//Modular shared methods make it a Modular
	Modular interface {
		//TopLevel dose this Module is top level,means should not be sub-module
		TopLevel() bool
		//PreLoad load as global Module
		PreLoad(l *LState)
		//PreloadSubModule use for submodule loading, Should NOT invoke manually
		PreloadSubModule(l *LState, t *LTable)
		//GetName the unique name (if is a Top Level Modular)
		GetName() string
	}
	fieldInfo struct {
		Help  string
		Value LValue
	}
	funcInfo struct {
		Help string
		Func LGFunction
	}
	//Module define a Module only contains Functions and value fields,maybe with submodules
	Module struct {
		Name       string               //Name of Modular
		Top        bool                 //is top level
		Help       string               //Help information of this Modular
		functions  map[string]funcInfo  //registered functions
		fields     map[string]fieldInfo //registered fields
		submodules []Modular            //registered sub modules
	}
	//Type define a LTable with MetaTable, which mimicry class like action in Lua
	Type struct {
		Module
		signature   reflect.Type              //reflect.Type
		constructor func(*LState) interface{} //Constructor for this Type , also can define other Constructor by add functions
		methods     map[string]funcInfo
		fields      map[string]fieldInfo
		override    map[Operate]funcInfo
	}
)

func helpFn(help map[string]string) LGFunction {
	key := make([]string, 0, len(help))
	for s := range help {
		key = append(key, s)
	}
	keys := strings.Join(key, ",")
	return func(s *LState) int {
		if s.GetTop() == 0 {
			s.Push(LString(keys))
		} else {
			s.Push(LString(help[s.ToString(1)]))
		}
		return 1
	}
}

//region Module

// NewModular create New Module
func NewModular(name string, help string, top bool) *Module {
	return &Module{Name: name, Help: help, Top: top}
}

func (m *Module) TopLevel() bool {
	return m.Top
}
func (m *Module) GetName() string {
	return m.Name
}
func (m *Module) PreLoad(l *LState) {
	if !m.Top {
		return
	}
	l.PreloadModule(m.Name, func(l *LState) int {
		mod := l.NewTable()
		fn := make(map[string]LGFunction)
		help := make(map[string]string)
		if m.Help != "" {
			help["?"] = m.Help
		}
		if len(m.functions) > 0 {
			for s, info := range m.functions {
				fn[s] = info.Func
				if info.Help != "" {
					help[s] = info.Help
				}
			}
		}
		if len(m.fields) > 0 {
			for key, value := range m.fields {
				l.SetField(mod, key, value.Value)
				if value.Help != "" {
					help[key+"?"] = value.Help
				}
			}
		}
		if len(m.submodules) > 0 {
			for _, t := range m.submodules {
				t.PreloadSubModule(l, mod)
			}
		}
		if len(help) > 0 {
			fn["Help"] = helpFn(help)
		}
		if len(fn) > 0 {
			l.SetFuncs(mod, fn)
		}
		l.Push(mod)
		return 1
	})
}
func (m *Module) PreloadSubModule(l *LState, t *LTable) {
	if m.Top {
		return
	}
	mod := l.NewTable()
	fn := make(map[string]LGFunction)
	help := make(map[string]string)
	if m.Help != "" {
		help["?"] = m.Help
	}
	if len(m.functions) > 0 {
		for s, info := range m.functions {
			fn[s] = info.Func
			if info.Help != "" {
				help[s] = info.Help
			}
		}
	}
	if len(m.fields) > 0 {
		for key, value := range m.fields {
			l.SetField(mod, key, value.Value)
			if value.Help != "" {
				help[key+"?"] = value.Help
			}
		}
	}
	if len(m.submodules) > 0 {
		for _, t := range m.submodules {
			t.PreloadSubModule(l, mod)
		}
	}
	if len(help) > 0 {
		fn["Help"] = helpFn(help)
	}
	if len(fn) > 0 {
		l.SetFuncs(mod, fn)
	}
	l.SetField(t, m.Name, mod)
}

// AddFunc add function to this Modular
//
// @name function name, must match lua limitation
//
// @help help string, if empty will not generate into help
//
// @fn the LGFunction
func (m *Module) AddFunc(name string, help string, fn LGFunction) *Module {
	if m.functions == nil {
		m.functions = make(map[string]funcInfo)
	} else if _, ok := m.functions[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.functions[name] = funcInfo{help, fn}
	return m

}

// AddField add value field to this Modular
//
// @name the field name
//
// @help help string, if empty will not generate into help
//
// @value the field value
func (m *Module) AddField(name string, help string, value LValue) *Module {
	if m.fields == nil {
		m.fields = make(map[string]fieldInfo)
	} else if _, ok := m.fields[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.fields[name] = fieldInfo{help, value}
	return m
}

// AddModule add sub-module to this Modular
//
// @mod the Module **Note** must with TopLevel false.
func (m *Module) AddModule(mod Modular) *Module {
	if mod.TopLevel() {
		panic(ErrIsTop)
	}
	m.submodules = append(m.submodules, mod)
	return m

}

//endregion

//region Type

// NewType create new Type
func NewType(name string, help string, top bool, ctor func(*LState) interface{}) *Type {
	return &Type{Module: Module{Name: name, Top: top, Help: help}, constructor: ctor}
}

// NewType2 create new Type with reflect Signature
func NewType2(sample interface{}, name string, help string, top bool, ctor func(*LState) interface{}) *Type {
	return &Type{Module: Module{Name: name, Top: top, Help: help}, signature: reflect.TypeOf(sample), constructor: ctor}
}

func (m *Type) PreLoad(l *LState) {
	if !m.Top {
		return
	}
	l.SetGlobal(m.Name, m.getOrBuildMeta(l))
}
func (m *Type) PreloadSubModule(l *LState, t *LTable) {
	if m.Top {
		return
	}
	t.RawSetString(m.Name, m.getOrBuildMeta(l))
}

func (m *Type) getOrBuildMeta(l *LState) *LTable {
	var mt *LTable
	var ok bool
	if mt, ok = l.GetTypeMetatable(m.Name).(*LTable); ok {
		return mt
	}
	mt = l.NewTypeMetatable(m.Name)
	fn := make(map[string]LGFunction)
	help := make(map[string]string)
	if m.Help != "" {
		help["?"] = m.Help
	}
	if m.constructor != nil {
		l.SetField(mt, "new", l.NewFunction(m.new))
	}
	if len(m.functions) > 0 {
		for s, info := range m.functions {
			fn[s] = info.Func
			if info.Help != "" {
				help[s] = info.Help
			}
		}
	}
	if len(m.fields) > 0 {
		for key, value := range m.fields {
			l.SetField(mt, key, value.Value)
			if value.Help != "" {
				help[key+"?"] = value.Help
			}
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
			if info.Help != "" {
				help[s] = info.Help
			}
		}
		// methods
		l.SetField(mt, "__index", l.SetFuncs(l.NewTable(), method))
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
			if info.Help != "" {
				help[name] = info.Help
			}
			l.SetField(mt, name, l.NewFunction(info.Func))
		}

	}
	if len(help) > 0 {
		fn["Help"] = helpFn(help)
	}
	if len(fn) > 0 {
		l.SetFuncs(mt, fn)
	}
	return mt
}

// New wrap an instance into LState
func (m Type) New(l *LState, val interface{}) int {
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, m.getOrBuildMeta(l))
	l.Push(ud)
	return 1
}
func (m Type) NewStoreState(l *StoredState, val interface{}) int {
	return m.New(l.LState, val)
}

// NewValue create new LValue
func (m Type) NewValue(l *LState, val interface{}) *LUserData {
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, m.getOrBuildMeta(l))
	return ud
}
func (m Type) NewValueStoreState(l *StoredState, val interface{}) *LUserData {
	return m.NewValue(l.LState, val)
}

// new internal creator
func (m Type) new(l *LState) int {
	val := m.constructor(l)
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, l.GetTypeMetatable(m.Name))
	l.Push(ud)
	return 1
}

// CanCast check the type can use cast (when construct with NewType2)
func (m Type) CanCast() bool {
	return m.signature != nil
}

// Cast nil if not current type else the value
func (m Type) Cast(u *LUserData) interface{} {
	if u == nil || reflect.TypeOf(u.Value) != m.signature {
		return nil
	}
	return u.Value
}

// AddMethod add method to this type which means instance method.
func (m *Type) AddMethod(name string, help string, value LGFunction) *Type {
	if m.methods == nil {
		m.methods = make(map[string]funcInfo)
	} else if _, ok := m.methods[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.methods[name] = funcInfo{help, value}
	return m
}

// AddMethod2 add method to this type which means instance method, with auto extract first argument.
func (m *Type) AddMethod2(name string, help string, act func(s *LState, data *LUserData) int) *Type {
	return m.AddMethod(name, help, func(s *LState) int {
		u := s.CheckUserData(1)
		if u == nil {
			return 0
		}
		return act(s, u)
	})
}

// AddMethod3 prechecked type (only create with NewType2).
func (m *Type) AddMethod3(name string, help string, act func(s *LState, data interface{}) int) *Type {
	if !m.CanCast() {
		panic("can't use AddMethod3 for not create with NewType2")
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
func (m *Type) Override(op Operate, help string, fn LGFunction) *Type {
	if m.override == nil {
		m.override = make(map[Operate]funcInfo)
	} else if _, ok := m.override[op]; ok {
		panic(ErrAlreadyExists)
	}
	m.override[op] = funcInfo{help, fn}
	return m
}

// Override2 see Override and AddMethod2
func (m *Type) Override2(op Operate, help string, act func(s *LState, data *LUserData) int) *Type {

	return m.Override(op, help, func(s *LState) int {
		u := s.CheckUserData(1)
		if u == nil {
			return 0
		}
		return act(s, u)
	})
}

// Override3 see Override and AddMethod3
func (m *Type) Override3(op Operate, help string, act func(s *LState, data interface{}) int) *Type {
	return m.Override(op, help, func(s *LState) int {
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

//endregion
