package glu

import (
	"github.com/ZenLiuCN/fn"
	lua "github.com/yuin/gopher-lua"
	"testing"
)

func TestGenericMod(t *testing.T) {
	mod := NewType[map[string]string]("textMap", `textMap module`, true, ``, func(state *lua.LState) (map[string]string, bool) {
		if state.GetTop() != 0 {
			state.RaiseError("no argument wanted")
			return nil, false
		}
		m := make(map[string]string)
		return m, true
	})
	fn.Panic(Register(mod))
	v := Get()
	defer v.Close()
	v.DoString(`
	t=require("textMap")
	print(t.help())
`)
}
