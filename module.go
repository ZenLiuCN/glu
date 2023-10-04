// Package glu  support yuin/gopher-lua with easy modular definition and other enchantments.
// glu.Modular and gua.BaseType will inject mod.Help(name string?) method to output HelpCache information.
// glu.Get: Pool function to get a lua.LState.
// glu.Put: Pool function to return a lua.LState.
// glu.registry: shared module registry.
// glu.Auto: config for autoload modules in registry into lua.LState.
package glu

import (
	. "github.com/yuin/gopher-lua"
	"strings"
)

type (
	Module interface {
		Modular
		// AddFunc add function to this Module
		//
		// @name function name, must match lua limitation
		//
		// @HelpCache HelpCache string, if empty will generate just Module.Function as HelpCache
		//
		// @fn the LGFunction
		AddFunc(name string, help string, fn LGFunction) Module

		// AddField add value field to this Module (static value)
		AddField(name string, help string, value LValue) Module
		// AddModule add submodule to this Module
		//
		// @mod the Mod , requires Mod.TopLevel is false.
		AddModule(mod Modular) Module
	}
	fieldInfo struct {
		Help  string
		Value LValue
	}
	funcInfo struct {
		Help string
		Func LGFunction
	}
	//Mod define a Mod only contains Functions and value fields,maybe with Submodules
	Mod struct {
		Name       string               //Name of Modular
		Top        bool                 //is top level
		Help       string               //Help information of this Modular
		functions  map[string]funcInfo  //registered functions
		fields     map[string]fieldInfo //registered fields
		Submodules []Modular            //registered sub modules
		prepared   bool                 //compute helper and other things, should just do once
		HelpCache  map[string]string    //exported helps for better use
	}
)

// NewModule create New Mod
func NewModule(name string, help string, top bool) *Mod {
	return &Mod{Name: name, Help: help, Top: top}
}

func (m *Mod) TopLevel() bool {
	return m.Top
}
func (m *Mod) GetName() string {
	return m.Name
}
func (m *Mod) GetHelp() string {
	return m.Help
}
func (m *Mod) prepare() {
	if m.prepared {
		return
	}
	help := make(map[string]string)
	mh := new(strings.Builder) //mod HelpCache builder
	if m.Help != "" {
		mh.WriteString(m.Help)
		mh.WriteRune('\n')
	} else {
		mh.WriteString(m.Name)
		mh.WriteRune('\n')
	}
	helpFuncReg(m.functions, help, mh, m.Name)
	helpFieldReg(m.fields, help, mh, m.Name)
	helpSubModReg(m.Submodules, help, mh, m.Name)

	if mh.Len() > 0 {
		help[HelpKey] = mh.String()
	}
	if EagerHelpPrepare && len(m.Submodules) > 0 {
		for _, sub := range m.Submodules {
			switch sub.(type) {
			case Prepare:
				sub.(Prepare).prepare()
			}
		}
	}
	m.HelpCache = help
	m.prepared = true
}
func (m *Mod) PreLoad(l *LState) {
	if !m.Top {
		return
	}
	m.prepare()
	l.PreloadModule(m.Name, func(l *LState) int {
		mod := l.NewTable()
		fn := make(map[string]LGFunction)
		if len(m.functions) > 0 {
			for s, info := range m.functions {
				fn[s] = info.Func
			}
		}
		if len(m.fields) > 0 {
			for key, value := range m.fields {
				l.SetField(mod, key, value.Value)
			}
		}
		if len(m.Submodules) > 0 {
			for _, t := range m.Submodules {
				t.PreloadSubModule(l, mod)
			}
		}
		if len(m.HelpCache) > 0 {
			fn[HelpFunc] = helpFn(m.HelpCache)
		}
		if len(fn) > 0 {
			l.SetFuncs(mod, fn)
		}
		l.Push(mod)
		return 1
	})
}
func (m *Mod) PreloadSubModule(l *LState, t *LTable) {
	if m.Top {
		return
	}
	m.prepare()
	mod := l.NewTable()
	fn := make(map[string]LGFunction)

	if len(m.functions) > 0 {
		for s, info := range m.functions {
			fn[s] = info.Func
		}
	}
	if len(m.fields) > 0 {
		for key, value := range m.fields {
			l.SetField(mod, key, value.Value)

		}
	}
	if len(m.Submodules) > 0 {
		for _, s := range m.Submodules {
			s.PreloadSubModule(l, mod)
		}
	}
	if len(m.HelpCache) > 0 {
		fn[HelpFunc] = helpFn(m.HelpCache)
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
// @HelpCache HelpCache string, if empty will not generate into HelpCache
//
// @fn the LGFunction
func (m *Mod) AddFunc(name string, help string, fn LGFunction) Module {
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
// @HelpCache HelpCache string, if empty will not generate into HelpCache
//
// @value the field value
func (m *Mod) AddField(name string, help string, value LValue) Module {
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
// @mod the Mod **Note** must with TopLevel false.
func (m *Mod) AddModule(mod Modular) Module {
	if mod.TopLevel() {
		panic(ErrIsTop)
	}
	m.Submodules = append(m.Submodules, mod)
	return m

}
