package gabs

import (
	. "github.com/Jeffail/gabs/v2"
	. "github.com/yuin/gopher-lua"
	"gua"
)

func init() {
	check := func(l *LState) *Container {
		ud := l.CheckUserData(1)
		if v, ok := ud.Value.(*Container); ok {
			return v
		}
		l.ArgError(1, "gabs expected")
		return nil
	}
	m := gua.NewModular("json", "json module", true)
	t := gua.NewType("json", false, "json type", func(*LState) interface{} {
		//TODO
		return New()
	}).
		AddMethod("string", "fetch json string", func(s *LState) int {
			v := check(s)
			s.Push(LString(v.String()))
			return 1
		}).
		AddMethod("string1", "fetch json string", func(s *LState) int {
			v := check(s)
			s.Push(LString(v.String()))
			return 1
		})
	m.AddModule(t)

	gua.Registry = append(gua.Registry, m)

}
