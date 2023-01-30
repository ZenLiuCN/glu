package glu

import lua "github.com/yuin/gopher-lua"

// Modular shared methods make it a Modular
type Modular interface {
	//TopLevel dose this Mod is top level,means should not be submodule
	TopLevel() bool
	//PreLoad load as global Mod
	PreLoad(l *lua.LState)
	//PreloadSubModule use for submodule loading, Should NOT invoke manually
	PreloadSubModule(l *lua.LState, t *lua.LTable)
	//GetName the unique name (if is a Top Level Modular)
	GetName() string
}
