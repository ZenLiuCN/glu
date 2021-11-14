package gua

import (
	"errors"
	. "github.com/yuin/gopher-lua"
)

var (
	Registry []Module
)

var (
	ErrAlreadyExists = errors.New("element already exists")
	ErrIsTop         = errors.New("element is top module")
)

type (
	Module interface {
		TopLevel() bool
		PreLoad(l *LState)
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
	Modular struct {
		Name      string
		Top       bool
		Help      string
		Func      map[string]funcInfo
		Fields    map[string]fieldInfo
		Submodule []Module
	}
	Type struct {
		Name      string
		Top       bool
		Help      string
		Ctor      func(*LState) interface{}
		Func      map[string]funcInfo
		Method    map[string]funcInfo
		Fields    map[string]fieldInfo
		Submodule []Module
	}
)

func helpFn(help map[string]string) LGFunction {
	return func(s *LState) int {
		if s.GetTop() == 0 {
			s.Push(LString(help["?"]))
		} else {
			s.Push(LString(help[s.ToString(1)]))
		}
		return 1
	}
}
func NewType(name string, top bool, help string, ctor func(*LState) interface{}) *Type {
	return &Type{Name: name, Top: top, Help: help, Ctor: ctor}
}

func NewModular(name string, help string, top bool) *Modular {
	return &Modular{Name: name, Help: help, Top: top}
}

func (m *Modular) TopLevel() bool {
	return m.Top
}
func (m *Modular) PreLoad(l *LState) {
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
		if len(m.Func) > 0 {
			for s, info := range m.Func {
				fn[s] = info.Func
				if info.Help != "" {
					help[s] = info.Help
				}
			}
		}
		if len(m.Fields) > 0 {
			for key, value := range m.Fields {
				l.SetField(mod, key, value.Value)
				if value.Help != "" {
					help[key+"?"] = value.Help
				}
			}
		}
		if len(m.Submodule) > 0 {
			for _, t := range m.Submodule {
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
func (m *Modular) PreloadSubModule(l *LState, t *LTable) {
	if m.Top {
		return
	}
	mod := l.NewTable()
	fn := make(map[string]LGFunction)
	help := make(map[string]string)
	if m.Help != "" {
		help["?"] = m.Help
	}
	if len(m.Func) > 0 {
		for s, info := range m.Func {
			fn[s] = info.Func
			if info.Help != "" {
				help[s] = info.Help
			}
		}
	}
	if len(m.Fields) > 0 {
		for key, value := range m.Fields {
			l.SetField(mod, key, value.Value)
			if value.Help != "" {
				help[key+"?"] = value.Help
			}
		}
	}
	if len(m.Submodule) > 0 {
		for _, t := range m.Submodule {
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

func (m *Modular) AddFunc(name string, help string, fn LGFunction) *Modular {
	if m.Func == nil {
		m.Func = make(map[string]funcInfo)
	} else if _, ok := m.Func[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.Func[name] = funcInfo{help, fn}
	return m

}
func (m *Modular) AddField(name string, help string, value LValue) *Modular {
	if m.Fields == nil {
		m.Fields = make(map[string]fieldInfo)
	} else if _, ok := m.Fields[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.Fields[name] = fieldInfo{help, value}
	return m
}
func (m *Modular) AddModule(mod Module) *Modular {
	if mod.TopLevel() {
		panic(ErrIsTop)
	}
	m.Submodule = append(m.Submodule, mod)
	return m

}

func (m *Type) TopLevel() bool {
	return m.Top
}
func (m *Type) PreLoad(l *LState) {
	if !m.Top {
		return
	}
	mt := l.NewTypeMetatable(m.Name)
	l.SetGlobal(m.Name, mt)
	fn := make(map[string]LGFunction)
	help := make(map[string]string)
	if m.Help != "" {
		help["?"] = m.Help
	}
	if m.Ctor != nil {
		l.SetField(mt, "new", l.NewFunction(m.new))
	}
	if len(m.Func) > 0 {
		for s, info := range m.Func {
			fn[s] = info.Func
			if info.Help != "" {
				help[s] = info.Help
			}
		}
	}
	if len(m.Fields) > 0 {
		for key, value := range m.Fields {
			l.SetField(mt, key, value.Value)
			if value.Help != "" {
				help[key+"?"] = value.Help
			}
		}
	}
	if len(m.Submodule) > 0 {
		for _, t := range m.Submodule {
			t.PreloadSubModule(l, mt)
		}
	}
	if len(m.Method) > 0 {
		method := make(map[string]LGFunction, len(m.Func))
		for s, info := range m.Method {
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

}
func (m Type) new(l *LState) int {
	val := m.Ctor(l)
	ud := l.NewUserData()
	ud.Value = val
	l.SetMetatable(ud, l.GetTypeMetatable(m.Name))
	l.Push(ud)
	return 1
}
func (m *Type) PreloadSubModule(l *LState, t *LTable) {
	if m.Top {
		return
	}
	mt := l.NewTypeMetatable(m.Name)
	t.RawSetString(m.Name, mt)
	fn := make(map[string]LGFunction)
	help := make(map[string]string)
	if m.Help != "" {
		help["?"] = m.Help
	}
	if m.Ctor != nil {
		l.SetField(mt, "new", l.NewFunction(m.new))
	}
	if len(m.Func) > 0 {
		for s, info := range m.Func {
			fn[s] = info.Func
			if info.Help != "" {
				help[s] = info.Help
			}
		}
	}
	if len(m.Fields) > 0 {
		for key, value := range m.Fields {
			l.SetField(mt, key, value.Value)
			if value.Help != "" {
				help[key+"?"] = value.Help
			}
		}
	}
	if len(m.Submodule) > 0 {
		for _, t := range m.Submodule {
			t.PreloadSubModule(l, mt)
		}
	}
	if len(m.Method) > 0 {
		method := make(map[string]LGFunction, len(m.Func))
		for s, info := range m.Method {
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
}
func (m *Type) AddFunc(name string, help string, fn LGFunction) *Type {
	if m.Func == nil {
		m.Func = make(map[string]funcInfo)
	} else if _, ok := m.Func[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.Func[name] = funcInfo{help, fn}
	return m
}
func (m *Type) AddField(name string, help string, value LValue) *Type {
	if m.Fields == nil {
		m.Fields = make(map[string]fieldInfo)
	} else if _, ok := m.Fields[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.Fields[name] = fieldInfo{help, value}
	return m
}
func (m *Type) AddMethod(name string, help string, value LGFunction) *Type {
	if m.Method == nil {
		m.Method = make(map[string]funcInfo)
	} else if _, ok := m.Method[name]; ok {
		panic(ErrAlreadyExists)
	}
	m.Method[name] = funcInfo{help, value}
	return m
}
func (m *Type) AddModule(mod Module) *Type {
	if mod.TopLevel() {
		panic(ErrIsTop)
	}
	m.Submodule = append(m.Submodule, mod)
	return m
}
