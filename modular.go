package glu

import (
	"errors"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"strings"
)

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
	//GetHelp Modular help info
	GetHelp() string
}

var (
	//HelpKey the module help key
	HelpKey = "?"
	//HelpFunc the help function name
	HelpFunc = "help"
	//HelpPrompt the prompt for no value supply for help
	HelpPrompt = "show help with those key word: "
	HelpChunk  = `chunk(code,name string)(Chunk?,string?) ==> pre compile string into bytecode`
	HelpHelp   = HelpFunc + `(topic string?)string? => fetch help of topic,'?' show topics,without topic show loadable modules`
	HelpTopic  = `?,chunk`
)

func helpFuncReg(fun map[string]funcInfo, helps map[string]string, mh *strings.Builder, mod string) {
	if len(fun) > 0 {
		for s, info := range fun {
			if info.Help != "" {
				helps[s] = fmt.Sprintf("%s.%s %s", mod, s, info.Help)
				mh.WriteString(fmt.Sprintf("%s.%s %s\n", mod, s, info.Help))
			} else {
				mh.WriteString(fmt.Sprintf("%s.%s\n", mod, s))
			}
		}
	}
}
func helpFieldReg(fun map[string]fieldInfo, helps map[string]string, mh *strings.Builder, mod string) {
	if len(fun) > 0 {
		for s, info := range fun {
			if info.Help != "" {
				helps[s] = fmt.Sprintf("%s.%s %s", mod, s, info.Help)
				mh.WriteString(fmt.Sprintf("%s.%s %s\n", mod, s, info.Help))
			} else {
				mh.WriteString(fmt.Sprintf("%s.%s\n", mod, s))
			}
		}
	}
}
func helpMethodReg(fun map[string]funcInfo, helps map[string]string, mh *strings.Builder, mod string) {
	if len(fun) > 0 {
		for s, info := range fun {
			if info.Help != "" {
				helps[s] = fmt.Sprintf("%s:%s %s", mod, s, info.Help)
				mh.WriteString(fmt.Sprintf("%s:%s %s\n", mod, s, info.Help))
			} else {
				mh.WriteString(fmt.Sprintf("%s:%s\n", mod, s))
			}
		}
	}
}
func helpSubModReg(fun []Modular, helps map[string]string, mh *strings.Builder, mod string) {
	if len(fun) > 0 {
		for _, info := range fun {
			s := info.GetName()
			if info.GetHelp() != "" {
				helps[s] = fmt.Sprintf("%s.%s %s", mod, s, info.GetHelp())
				mh.WriteString(fmt.Sprintf("%s.%s %s\n", mod, s, info.GetHelp()))
			} else {
				mh.WriteString(fmt.Sprintf("%s.%s\n", mod, s))
			}
		}
	}
}
func helpOperatorReg(operators map[Operate]funcInfo, hasMethods bool, helps map[string]string, mh *strings.Builder, mod string) {
	if len(operators) > 0 {
		for op, info := range operators {
			var sym string
			var name string
			switch op {
			case OPERATE_ADD:
				name = "__add"
				sym = "+"
			case OPERATE_SUB:
				name = "__sub"
				sym = "-"
			case OPERATE_MUL:
				name = "__mul"
				sym = "*"
			case OPERATE_DIV:
				name = "__div"
				sym = "/"
			case OPERATE_UNM:
				name = "__unm"
				sym = "-"
			case OPERATE_MOD:
				name = "__mod"
				sym = "%"
			case OPERATE_POW:
				name = "__pow"
				sym = "^"
			case OPERATE_CONCAT:
				name = "__concat"
				sym = ".."
			case OPERATE_EQ:
				name = "__eq"
				sym = "=="
			case OPERATE_LT:
				name = "__lt"
				sym = "<"
			case OPERATE_LE:
				name = "__le"
				sym = "<="
			case OPERATE_LEN:
				name = "__len"
				sym = "#"
			case OPERATE_NEWINDEX:
				name = "__newindex"
				sym = "[]="
			case OPERATE_TO_STRING:
				name = "__to_string"
				sym = "tostring"
			case OPERATE_CALL:
				name = "__call"
				sym = "()"
			case OPERATE_INDEX:
				if hasMethods {
					panic(ErrIndexOverrideWithMethods)
				}
				name = "__index"
				sym = "[]"
			default:
				panic(fmt.Errorf("unsupported operators of %d", op))
			}
			if info.Help != "" {
				helps[name] = fmt.Sprintf("%s::%s %s\n", mod, sym, info.Help)
				mh.WriteString(fmt.Sprintf("%s::%s %s\n", mod, sym, info.Help))
			} else {
				mh.WriteString(fmt.Sprintf("%s::%s\n", mod, sym))
			}
		}

	}
}
func helpCtorReg(ctor func(*lua.LState) interface{}, ctor2 string, help map[string]string, mh *strings.Builder, name string) {
	if ctor != nil {
		if ctor2 != "" {
			help["new"] = fmt.Sprintf("%s.new %s", name, ctor2)
		}
		mh.WriteString(fmt.Sprintf("%s.new %s\n", name, ctor2))
	}
}
func helpFn(help map[string]string) lua.LGFunction {
	key := make([]string, 0, len(help))
	for s := range help {
		key = append(key, s)
	}
	keys := HelpPrompt + strings.Join(key, ",")
	return func(s *lua.LState) int {
		if s.GetTop() == 0 {
			s.Push(lua.LString(keys))
		} else {
			s.Push(lua.LString(help[s.ToString(1)]))
		}
		return 1
	}
}

var (
	ErrAlreadyExists            = errors.New("element already exists")
	ErrIndexOverrideWithMethods = errors.New("element both have methods and index overrides")
	ErrIsTop                    = errors.New("element is top module")
)
