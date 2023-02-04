package glu

import (
	. "github.com/yuin/gopher-lua"
	"strings"
)

var (
	//GluMod the global module
	GluMod = glu(0)
	//HelpKey the module help key
	HelpKey = "?"
	//HelpFunc the help function name
	HelpFunc = "help"
	//HelpPrompt the prompt for no value supply for help
	HelpPrompt = "show help with those key word:"
	HelpChunk  = `chunk(code,name string)(Chunk?,string?) ==> pre compile string into bytecode`
	HelpHelp   = HelpFunc + `(topic string?)string? ==> fetch help of topic`
	HelpTopic  = `?,chunk`
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
	l.SetGlobal(HelpFunc, l.NewFunction(func(s *LState) int {
		if s.GetTop() == 0 {
			s.Push(LString(HelpPrompt + HelpTopic))
			return 1
		}
		topic := s.CheckString(1)
		switch topic {
		case HelpKey:
			s.Push(LString(HelpHelp))
		case "chunk":
			s.Push(LString(HelpChunk))
		default:
			s.Push(LNil)
		}
		return 1
	}))
}

func (c glu) PreloadSubModule(l *LState, t *LTable) {
	panic("implement me")
}

func helpFn(help map[string]string) LGFunction {
	key := make([]string, 0, len(help))
	for s := range help {
		key = append(key, s)
	}
	keys := HelpPrompt + strings.Join(key, ",")
	return func(s *LState) int {
		if s.GetTop() == 0 {
			s.Push(LString(keys))
		} else {
			s.Push(LString(help[s.ToString(1)]))
		}
		return 1
	}
}
