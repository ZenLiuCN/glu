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
	t := gua.NewType("json", true, "json module", func(*LState) interface{} {
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

	gua.Registry = append(gua.Registry, t)

}
