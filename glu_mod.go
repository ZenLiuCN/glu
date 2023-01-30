package glu

import . "github.com/yuin/gopher-lua"

var (
	//GluMod the global module
	GluMod = glu(0)
)

const (
	helpChunk = `chunk(code,name string)(Chunk?,string?) ==> pre compile string into bytecode`
	helpHelp  = `Help(topic string?)string? ==> fetch help of topic`
	helpTopic = `?,chunk`
)

type (
	Chunk = *FunctionProto
	glu   int
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
	l.SetGlobal("chunk", l.NewFunction(func(s *LState) int {
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
	}))
	l.SetGlobal("Help", l.NewFunction(func(s *LState) int {
		if s.GetTop() == 0 {
			s.Push(LString(helpTopic))
			return 1
		}
		topic := s.CheckString(1)
		switch topic {
		case "?":
			s.Push(LString(helpHelp))
		case "chunk":
			s.Push(LString(helpChunk))
		default:
			s.Push(LNil)
		}
		return 1
	}))
}

func (c glu) PreloadSubModule(l *LState, t *LTable) {
	panic("implement me")
}
