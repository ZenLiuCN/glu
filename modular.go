// Package glu  support yuin/gopher-lua with easy modular definition and other enchantments.
// glu.Modular and gua.Type will inject mod.Help(name string?) method to output help information.
// glu.Get: Pool function to get a lua.LState.
// glu.Put: Pool function to return a lua.LState.
// glu.Registry: shared module registry.
// glu.Auto: config for autoload modules in Registry into lua.LState.
//
package glu

import (
	"errors"
	. "github.com/yuin/gopher-lua"
	"strings"
)

var (
	Registry []Modular
)

var (
	ErrAlreadyExists = errors.New("element already exists")
	ErrIsTop         = errors.New("element is top module")
)

type (
	//Modular shared methods make it a Modular
	Modular interface {
		//TopLevel dose this Module is top level,means should not be sub-module
		TopLevel() bool
		//PreLoad load as global Module
		PreLoad(l *LState)
		//PreloadSubModule use for sub-module loading, Should NOT invoke manually
		PreloadSubModule(l *LState, t *LTable)
	}
	fieldInfo struct {
		Help  string
		Value LValue
	}
	funcInfo struct {
		Help string
		Func LGFunction
	}
	//Module define a Module only contains Functions and value fields,maybe with sub-modules
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
		Name        string                    //Name of Type, it's also the MetaTable Name
		Top         bool                      //is top level
		Help        string                    //Help information of this Type
		constructor func(*LState) interface{} //Constructor for this Type , also can define other Constructor by add functions
		functions   map[string]funcInfo
		methods     map[string]funcInfo
		fields      map[string]fieldInfo
		submodules  []Modular
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
func NewType(name string, help string, top bool, ctor func(*LState) interface{}) *Type {
	return &Type{Name: name, Top: top, Help: help, constructor: ctor}
}
func NewModular(name string, help string, top bool) *Module {
	return &Module{Name: name, Help: help, Top: top}
}

func (m *Module) TopLevel() bool {
	return m.Top
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

//AddFunc add function to this Modular
//
//@name function name, must match lua limitation
//
//@help help string, if empty will not generate into help
//
//@fn the LGFunction
func (m *Module) AddFunc(name string, help string, fn LGFunction) *Module {
	if m.functions == nil {
		m.functions = make(map[string]funcInfo)
	} else if _, ok := m.functions[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.functions[name] = funcInfo{help, fn}
	return m

}

//AddField add value field to this Modular
//
//@name the field name
//
//@help help string, if empty will not generate into help
//
//@value the field value
func (m *Module) AddField(name string, help string, value LValue) *Module {
	if m.fields == nil {
		m.fields = make(map[string]fieldInfo)
	} else if _, ok := m.fields[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.fields[name] = fieldInfo{help, value}
	return m
}

//AddModule add sub-module to this Modular
//
//@mod the Module **Note** must with TopLevel false.
func (m *Module) AddModule(mod Modular) *Module {
	if mod.TopLevel() {
		panic(ErrIsTop)
	}
	m.submodules = append(m.submodules, mod)
	return m

}

func (m *Type) TopLevel() bool {
	return m.Top
}

//PreLoad Load as global module
func (m *Type) PreLoad(l *LState) {
	if !m.Top {
		return
	}
	l.SetGlobal(m.Name, m.getOrBuildMeta(l))
}

//PreloadSubModule submodule loading should not call be manual
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
	if len(help) > 0 {
		fn["Help"] = helpFn(help)
	}
	if len(fn) > 0 {
		l.SetFuncs(mt, fn)
	}
	return mt
}

//New wrap an instance into LState
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

//NewValue create new LValue
func (m Type) NewValue(l *LState, val interface{}) *LUserData {
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, m.getOrBuildMeta(l))
	return ud
}
func (m Type) NewValueStoreState(l *StoredState, val interface{}) *LUserData {
	return m.NewValue(l.LState, val)
}

//new internal creator
func (m Type) new(l *LState) int {
	val := m.constructor(l)
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, l.GetTypeMetatable(m.Name))
	l.Push(ud)
	return 1
}

//AddFunc add function to this type
func (m *Type) AddFunc(name string, help string, fn LGFunction) *Type {
	if m.functions == nil {
		m.functions = make(map[string]funcInfo)
	} else if _, ok := m.functions[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.functions[name] = funcInfo{help, fn}
	return m
}

//AddField add value field to this type
func (m *Type) AddField(name string, help string, value LValue) *Type {
	if m.fields == nil {
		m.fields = make(map[string]fieldInfo)
	} else if _, ok := m.fields[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.fields[name] = fieldInfo{help, value}
	return m
}

//AddMethod add method to this type which means instance method.
func (m *Type) AddMethod(name string, help string, value LGFunction) *Type {
	if m.methods == nil {
		m.methods = make(map[string]funcInfo)
	} else if _, ok := m.methods[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.methods[name] = funcInfo{help, value}
	return m
}

//AddModule add sub-module to this type
func (m *Type) AddModule(mod Modular) *Type {
	if mod.TopLevel() {
		panic(ErrIsTop)
	}
	m.submodules = append(m.submodules, mod)
	return m
}
