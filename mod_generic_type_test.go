package glu

import (
	"github.com/ZenLiuCN/fn"
	lua "github.com/yuin/gopher-lua"
	"testing"
)

func init() {
	mod := NewType[map[string]string]("map", `textMap module`, true, ``, func(state *lua.LState) map[string]string {
		if state.GetTop() != 0 {
			state.RaiseError("no argument wanted")
		}
		m := make(map[string]string)
		return m
	})
	fn.Panic(Register(mod))
}
func TestGenericMod(t *testing.T) {

	v := Get()
	defer v.Close()
	err := v.DoString(`
	print(help())
	print(map.help())
	res,info=pcall(map.new,1)
	print(res)
	print(info)
`)
	if err != nil {
		t.Fatal(err)
	}
}
