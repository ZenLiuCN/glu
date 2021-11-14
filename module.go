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
		Func      map[string]funcInfo
		Fields    map[string]fieldInfo
		Submodule []Module
	}
	Type struct {
		Name      string
		Top       bool
		Func      map[string]funcInfo
		Method    map[string]funcInfo
		Ctor      func(*LState) interface{}
		Fields    map[string]fieldInfo
		Submodule []Module
	}
)

func (m *Modular) TopLevel() bool {
	return m.Top
}
func (m *Modular) PreLoad(l *LState) {
	if !m.Top {
		return
	}
	l.PreloadModule(m.Name, func(l *LState) int {
		mod := l.NewTable()
		if len(m.Func) > 0 {
			fn := make(map[string]LGFunction, len(m.Func)*2)
			for s, info := range m.Func {
				fn[s] = info.Func
				if info.Help != "" {
					fn[s+"?"] = func(l *LState) int {
						l.Push(LString(info.Help))
						return 1
					}
				}
			}
			l.SetFuncs(mod, fn)
		}
		if len(m.Fields) > 0 {
			for key, value := range m.Fields {
				l.SetField(mod, key, value.Value)
				if value.Help != "" {
					l.SetField(mod, key+"?", LString(value.Help))
				}
			}
		}
		if len(m.Submodule) > 0 {
			for _, t := range m.Submodule {
				t.PreloadSubModule(l, mod)
			}
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
	if len(m.Func) > 0 {
		fn := make(map[string]LGFunction, len(m.Func)*2)
		for s, info := range m.Func {
			fn[s] = info.Func
			if info.Help != "" {
				fn[s+"?"] = func(l *LState) int {
					l.Push(LString(info.Help))
					return 1
				}
			}
		}
		l.SetFuncs(mod, fn)
	}
	if len(m.Fields) > 0 {
		for key, value := range m.Fields {
			l.SetField(mod, key, value.Value)
			if value.Help != "" {
				l.SetField(mod, key+"?", LString(value.Help))
			}
		}
	}
	if len(m.Submodule) > 0 {
		for _, t := range m.Submodule {
			t.PreloadSubModule(l, mod)
		}
	}
	l.SetField(t, m.Name, mod)
}

func (m *Modular) AddFunc(name string, help string, fn LGFunction) error {
	if m.Func == nil {
		m.Func = make(map[string]funcInfo)
	} else if _, ok := m.Func[name]; ok {
		return ErrAlreadyExists
	}
	m.Func[name] = funcInfo{help, fn}
	return nil
}
func (m *Modular) AddField(name string, help string, value LValue) error {
	if m.Fields == nil {
		m.Fields = make(map[string]fieldInfo)
	} else if _, ok := m.Fields[name]; ok {
		return ErrAlreadyExists
	}
	m.Fields[name] = fieldInfo{help, value}
	return nil
}
func (m *Modular) AddModule(mod Module) error {
	if mod.TopLevel() {
		return ErrIsTop
	}
	m.Submodule = append(m.Submodule, mod)
	return nil
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
	if m.Ctor != nil {
		l.SetField(mt, "new", l.NewFunction(m.new))
	}
	if len(m.Func) > 0 {
		fn := make(map[string]LGFunction, len(m.Func)*2)
		for s, info := range m.Func {
			fn[s] = info.Func
			if info.Help != "" {
				fn[s+"?"] = func(l *LState) int {
					l.Push(LString(info.Help))
					return 1
				}
			}
		}
		l.SetFuncs(mt, fn)
	}
	if len(m.Fields) > 0 {
		for key, value := range m.Fields {
			l.SetField(mt, key, value.Value)
			if value.Help != "" {
				l.SetField(mt, key+"?", LString(value.Help))
			}
		}
	}
	if len(m.Submodule) > 0 {
		for _, t := range m.Submodule {
			t.PreloadSubModule(l, mt)
		}
	}
	if len(m.Method) > 0 {
		fn := make(map[string]LGFunction, len(m.Func))
		hlp := make(map[string]LGFunction, len(m.Func))
		for s, info := range m.Method {
			fn[s] = info.Func
			if info.Help != "" {
				hlp[s+"?"] = func(l *LState) int {
					l.Push(LString(info.Help))
					return 1
				}
			}
		}
		// methods
		l.SetField(mt, "__index", l.SetFuncs(l.NewTable(), fn))
		l.SetFuncs(mt, hlp)
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
	if m.Ctor != nil {
		l.SetField(mt, "new", l.NewFunction(m.new))
	}
	if len(m.Func) > 0 {
		fn := make(map[string]LGFunction, len(m.Func)*2)
		for s, info := range m.Func {
			fn[s] = info.Func
			if info.Help != "" {
				fn[s+"?"] = func(l *LState) int {
					l.Push(LString(info.Help))
					return 1
				}
			}
		}
		l.SetFuncs(mt, fn)
	}
	if len(m.Fields) > 0 {
		for key, value := range m.Fields {
			l.SetField(mt, key, value.Value)
			if value.Help != "" {
				l.SetField(mt, key+"?", LString(value.Help))
			}
		}
	}
	if len(m.Submodule) > 0 {
		for _, t := range m.Submodule {
			t.PreloadSubModule(l, mt)
		}
	}
	if len(m.Method) > 0 {
		fn := make(map[string]LGFunction, len(m.Func))
		hlp := make(map[string]LGFunction, len(m.Func))
		for s, info := range m.Method {
			fn[s] = info.Func
			if info.Help != "" {
				hlp[s+"?"] = func(l *LState) int {
					l.Push(LString(info.Help))
					return 1
				}
			}
		}
		// methods
		l.SetField(mt, "__index", l.SetFuncs(l.NewTable(), fn))
		l.SetFuncs(mt, hlp)
	}
}
func (m *Type) AddFunc(name string, help string, fn LGFunction) error {
	if m.Func == nil {
		m.Func = make(map[string]funcInfo)
	} else if _, ok := m.Func[name]; ok {
		return ErrAlreadyExists
	}
	m.Func[name] = funcInfo{help, fn}
	return nil
}
func (m *Type) AddField(name string, help string, value LValue) error {
	if m.Fields == nil {
		m.Fields = make(map[string]fieldInfo)
	} else if _, ok := m.Fields[name]; ok {
		return ErrAlreadyExists
	}
	m.Fields[name] = fieldInfo{help, value}
	return nil
}
func (m *Type) AddMethod(name string, help string, value LGFunction) error {
	if m.Method == nil {
		m.Method = make(map[string]funcInfo)
	} else if _, ok := m.Method[name]; ok {
		return ErrAlreadyExists
	}
	m.Method[name] = funcInfo{help, value}
	return nil
}
func (m *Type) AddModule(mod Module) error {
	if mod.TopLevel() {
		return ErrIsTop
	}
	m.Submodule = append(m.Submodule, mod)
	return nil
}
