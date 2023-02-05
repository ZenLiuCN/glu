package glu

import (
	. "github.com/yuin/gopher-lua"
	"strings"
)

var (
	//BaseMod the global module
	BaseMod = glu(map[string]string{})
)

type (
	Chunk = *FunctionProto
	glu   map[string]string
)

func (c glu) TopLevel() bool {
	return true
}
func (c glu) CheckChunk(s *LState, n int) Chunk {
	ud := s.CheckUserData(n)
	if v, ok := ud.Value.(Chunk); ok {
		return v
	}
	s.ArgError(n, "chunk expected")
	return nil
}
func (c glu) PreLoad(l *LState) {
	l.SetGlobal("chunk", l.NewFunction(SafeFunc(func(s *LState) int {
		chunk, err := CompileChunk(s.CheckString(1), s.CheckString(2))
		if err != nil {
			s.Push(LNil)
			s.Push(LString(err.Error()))
			return 2
		}
		ud := s.NewUserData()
		ud.Value = chunk
		s.Push(ud)
		s.Push(LNil)
		return 2
	})))
	l.SetGlobal(HelpFunc, l.NewFunction(SafeFunc(func(s *LState) int {
		if s.GetTop() < 1 {
			if i, ok := c["mod"]; ok {
				s.Push(LString(i))
				return 1
			}
			sub := new(strings.Builder)
			sub.WriteString(HelpHelp)
			sub.WriteString("\nExists loadable modules:\n")
			l.G.Global.RawGetString("package").(*LTable).RawGetString("preload").(*LTable).ForEach(func(k LValue, _ LValue) {
				sub.WriteString(k.String() + " module \n")
			})
			i := sub.String()
			s.Push(LString(i))
			c["mod"] = i
			return 1
		}
		t := s.CheckString(1)
		switch t {
		case HelpKey:
			s.Push(LString(HelpTopic))
		case "chunk":
			s.Push(LString(HelpChunk))
		default:
			s.Push(LNil)
		}
		return 1
	})))
}
