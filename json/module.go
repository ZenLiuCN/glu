package json

import (
	"fmt"
	. "github.com/Jeffail/gabs/v2"
	"github.com/ZenLiuCN/fn"
	. "github.com/ZenLiuCN/glu/v3"
	. "github.com/yuin/gopher-lua"
	"go/types"
)

var (
	JSON   Type[*Container]
	MODULE Module
)

func init() {

	checkPath := func(l *LState, c *Container, args int) (*Container, string) {
		if c == nil {
			return nil, ""
		}
		if l.GetTop() == 2 && args == 0 {
			return c.Path(l.ToString(2)), l.ToString(2)
		} else if l.GetTop() == args+1 {
			return c, ""
		} else if l.GetTop() == args+2 {
			p := l.ToString(2)
			if p == "" {
				return c, ""
			}
			return c.Path(p), p
		}
		return nil, ""
	}
	MODULE = NewModule("json", `json is wrapper of jeffail/gabs as dynamic json tool.`, true).
		AddFunc("of", `(table|number|string|boolean)JSON		create json from value`,
			func(s *LState) int {
				s.CheckTypes(1, LTString, LTNumber, LTBool, LTTable)
				v := s.Get(1)
				switch v.Type() {
				case LTString:
					g := New()
					_, _ = g.Set(s.ToString(1))
					return JSON.New(s, g)
				case LTNumber:
					g := New()
					_, _ = g.Set(float64(s.ToNumber(1)))
					return JSON.New(s, g)
				case LTBool:
					g := New()
					_, _ = g.Set(s.ToBool(1))
					return JSON.New(s, g)
				case LTTable:
					return JSON.New(s, parseTable(s, s.ToTable(1), New()))
				default:
					s.ArgError(1, "invalid")
					return 0
				}
			}).
		AddFunc("stringify", `(JSON)string		convert json to string`,
			func(s *LState) int {
				v := JSON.Check(s, 1)

				s.Push(LString(v.String()))
				return 1
			}).
		AddFunc("parse", `(string)JSON		create json from string value`,
			func(s *LState) int {
				v := s.CheckString(1)
				g := fn.Panic1(ParseJSON([]byte(v)))
				return JSON.New(s, g)
			})
	JSON = NewTypeCast(func(a any) (v *Container, ok bool) { v, ok = a.(*Container); return }, "JSON", `json.JSON`, false, `(string?)JSON? 	 create JSON instance.`,
		func(s *LState) *Container {
			if s.GetTop() == 1 {
				v, err := ParseJSON([]byte(s.CheckString(1)))
				if err != nil {
					s.ArgError(1, "invalid JSON string")
				}
				return v
			} else if s.GetTop() != 0 {
				s.RaiseError("bad argument for create JSON")
			}
			return New()
		}).
		AddMethodCast("json", `(pretty boolean=false,ident string='\t')string 	 json string of the JSON, if empty will be '{}'.`,
			func(s *LState, v *Container) int {
				if s.GetTop() >= 2 {
					s.CheckType(2, LTBool)
					if s.ToBool(2) {
						n := "\t"
						if s.GetTop() == 3 {
							s.CheckType(3, LTString)
							n = s.ToString(3)
						}
						s.Push(LString(v.StringIndent("", n)))
						return 1
					}
				}
				s.Push(LString(v.String()))
				return 1
			}).
		AddMethodCast("path", `(string?)JSON?  	 fetch JSON by path, path is gabs path.`,
			func(s *LState, v *Container) int {
				p := s.CheckString(2)
				if v.ExistsP(p) {
					return JSON.New(s, v.Path(p))
				}
				s.Push(LNil)
				return 1
			}).
		AddMethodCast("exists", `(string?)boolean  	 check existence of path.`,
			func(s *LState, v *Container) int {
				p, _ := checkPath(s, v, 0)
				s.Push(LBool(p != nil))
				return 1
			}).
		AddMethodCast("get", `(int|string)JSON?  	 fetch value at index for array or key for object.`,
			func(s *LState, v *Container) int {
				s.CheckTypes(2, LTString, LTNumber)
				p := s.Get(2)
				if p.Type() == LTString {
					x := p.String()
					if v.ExistsP(x) {
						return JSON.New(s, v.Path(x))
					} else {
						s.Push(LNil)
						return 1
					}
				} else if p.Type() == LTNumber {
					i := int(p.(LNumber))
					x := v.Index(i)
					if x == nil {
						s.Push(LNil)
						return 1
					}
					return JSON.New(s, x)
				}
				return 0
			}).
		AddMethodCast("type", `()int  	 fetch JSON type: 0 nil,1 string,2 number,3 boolean,4 array,5 object.`,
			func(s *LState, v *Container) int {
				if _, ok := v.Data().(string); ok {
					s.Push(LNumber(1))
				} else if _, ok = v.Data().(float64); ok {
					s.Push(LNumber(2))
				} else if _, ok = v.Data().(bool); ok {
					s.Push(LNumber(3))
				} else if _, ok = v.Data().([]any); ok {
					s.Push(LNumber(4))
				} else if _, ok = v.Data().(map[string]any); ok {
					s.Push(LNumber(5))
				} else {
					s.Push(LNumber(0))
				}
				return 1
			}).
		AddMethodCast("set", `(string?, JSON|string|number|bool|nil)string?  	 set value at path.if value is nil,will delete it.this can't append array.'`,
			func(s *LState, v *Container) int {
				idx := 2
				p := ""
				if s.GetTop() == 2 {
					p = s.ToString(2)
				} else if s.GetTop() == 3 {
					p = s.ToString(2)
					idx = 3
				}
				s.CheckTypes(idx, LTUserData, LTString, LTNumber, LTBool, LTNil)
				x := s.Get(idx)
				val, ok := unpack(x)
				var err error
				if !ok {
					s.ArgError(idx, fmt.Sprintf("invalid data type of %T", x))
					return 0
				} else if val == LNil {
					if p == "" {
						return 0
					}
					if p != "" && !v.ExistsP(p) {
						s.Push(LString("path not exist"))
						return 1
					}
					err = v.DeleteP(p)
					if err != nil {
						s.Push(LString("set value fail:" + err.Error()))
						return 1
					}
					return 0
				}
				if p == "" {
					_, err = v.Set(val)
					if err != nil {
						s.Push(LString("set value fail:" + err.Error()))
						return 1
					}
					return 0
				}
				_, err = v.SetP(val, p)
				if err != nil {
					s.Push(LString("set value fail:" + err.Error()))
					return 1
				}
				return 0
			}).
		AddMethodCast("append", `(string?, JSON|string|number|bool|nil)string?  	 append value, path must pointer to array.`,
			func(s *LState, c *Container) int {
				_, p := checkPath(s, c, 1)
				var idx int
				if p != "" {
					idx = 3
				} else {
					idx = 2
				}
				s.CheckTypes(idx, LTUserData, LTString, LTNumber, LTBool, LTNil)
				x := s.Get(idx)
				val, ok := unpack(x)
				if !ok {
					s.ArgError(idx, "invalid data type")
				} else if val == LNil {
					val = nil
				}
				var err error
				if p == "" {
					if m, ok := c.Data().(map[string]any); ok && len(m) > 0 {
						s.Push(LString("value at path is object"))
						return 1
					} else if ok {
						_, err = c.Array()
						if err != nil {
							s.Push(LString("append value fail:" + err.Error()))
							return 1
						}
					}
					if val == nil {
						idx = len(c.Data().([]any)) - 1
						if idx < 0 {
							return 0
						}
						err = c.ArrayRemove(idx)
					} else {
						err = c.ArrayAppend(val)
					}
					if err != nil {
						s.Push(LString("append value fail:" + err.Error()))
						return 1
					}
					return 0
				} else if !c.ExistsP(p) {
					if val == nil {
						return 0
					}
					err = c.ArrayAppendP(val, p)
					if err != nil {
						s.Push(LString("append value fail:" + err.Error()))
						return 1
					}
					return 0
				} else if m, ok := c.Path(p).Data().(map[string]any); ok && len(m) > 0 {
					s.Push(LString("value at path is object"))
					return 1
				} else if ok {
					_, err = c.Set([]any{}, p)
					if err != nil {
						s.Push(LString("append value fail:" + err.Error()))
						return 1
					}
				}
				if val == nil {
					idx = len(c.Path(p).Data().([]any)) - 1
					if idx < 0 {
						return 0
					}
					err = c.ArrayRemoveP(idx, p)
				} else {
					err = c.ArrayAppendP(val, p)
				}
				if err != nil {
					s.Push(LString("append value fail:" + err.Error()))
					return 1
				}
				return 0
			}).
		AddMethodCast("isArray", `(string?)bool  	 check if it's array at path.`,
			func(s *LState, c *Container) int {
				v, _ := checkPath(s, c, 0)
				if v == nil {
					s.Push(LFalse)
				} else {
					if _, ok := v.Data().([]any); ok {
						s.Push(LTrue)
					} else {
						s.Push(LFalse)
					}
				}
				return 1
			}).
		AddMethodCast("isObject", `(string?)bool  	 check if it's object at path.`,
			func(s *LState, c *Container) int {
				v, _ := checkPath(s, c, 0)
				if v == nil {
					s.Push(LFalse)
				} else if _, ok := v.Data().(map[string]any); ok {
					s.Push(LTrue)
				} else {
					s.Push(LFalse)
				}

				return 1
			}).
		AddMethodCast("bool", `(string?)bool  	 fetch value as boolean, if not exists return false.`,
			func(s *LState, c *Container) int {
				v, _ := checkPath(s, c, 0)
				if v == nil {
					s.Push(LFalse)
				} else if b, ok := v.Data().(bool); !ok {
					s.Push(LFalse)
				} else if b {
					s.Push(LTrue)
				} else {
					s.Push(LFalse)
				}
				return 1
			}).
		AddMethodCast("string", `(string?)string?  	 fetch value as string, if not exists or not string return nil.`,
			func(s *LState, c *Container) int {
				v, _ := checkPath(s, c, 0)
				if v == nil {
					s.Push(LNil)
				} else if b, ok := v.Data().(string); !ok {
					if _, ok := v.Data().([]byte); ok {
						ts := v.String()
						if len(ts) > 2 {
							ts = ts[1 : len(ts)-1]
						}
						s.Push(LString(ts))
						return 1
					}
					s.Push(LNil)
				} else {
					s.Push(LString(b))
				}
				return 1
			}).
		AddMethodCast("number", `(string?)number?  	 fetch value as number, if not exists return nil.`,
			func(s *LState, c *Container) int {
				v, _ := checkPath(s, c, 0)
				if v == nil {
					s.Push(LNil)
				} else if b, ok := v.Data().(float64); !ok {
					s.Push(LNil)
				} else {
					s.Push(LNumber(b))
				}
				return 1
			}).
		AddMethodCast("size", `(string?)number?  	 fetch  object size or array size else nil.`,
			func(s *LState, c *Container) int {
				v, _ := checkPath(s, c, 0)
				if b, ok := v.Data().(map[string]any); ok {
					s.Push(LNumber(len(b)))
				} else if a, ok := v.Data().([]any); ok {
					s.Push(LNumber(len(a)))
				} else if v == nil {
					s.Push(LNil)
				} else {
					s.Push(LNumber(1))
				}
				return 1
			}).
		AddMethodCast("raw", `(string?)nil|string|number|table?  	 convert to table.`,
			func(s *LState, c *Container) int {
				v, _ := checkPath(s, c, 0)
				s.Push(pack(v, s))
				return 1
			}).
		OverrideCast(OPERATE_TOSTRING, `same as JSON:json()`, func(s *LState, i *Container) int {
			s.Push(LString(i.String()))
			return 1
		})

	fn.Panic(Register(MODULE.AddModule(JSON)))

}
func parseTable(s *LState, t *LTable, g *Container) *Container {
	arr := t.MaxN() != 0 && t.MaxN() == t.Len()
	if arr {
		if _, ok := g.Data().(map[string]any); ok {
			_, err := g.Array()
			if err != nil {
				panic(err)
			}
		}
	}
	t.ForEach(func(k LValue, v LValue) {
		switch v.Type() {
		case LTString:
			if arr {
				_ = g.ArrayAppend(v.String())
			} else {
				_, _ = g.Set(v.String(), k.String())
			}
		case LTNumber:
			if arr {
				_ = g.ArrayAppend(float64(v.(LNumber)))
			} else {
				_, _ = g.Set(float64(v.(LNumber)), k.String())
			}
		case LTBool:
			n := v == LTrue
			if arr {
				_ = g.ArrayAppend(n)
			} else {
				_, _ = g.Set(n, k.String())
			}
		case LTTable:
			o := New()
			parseTable(s, v.(*LTable), o)
			if arr {
				_ = g.ArrayAppend(o)
			} else {
				_, _ = g.Set(o, k.String())
			}
		default:
			s.RaiseError("unsupported type")
		}
	})
	return g
}
func unpack(v LValue) (any, bool) {
	switch v.Type() {
	case LTString:
		return v.String(), true
	case LTNumber:
		return float64(v.(LNumber)), true
	case LTBool:
		return v == LTrue, true
	case LTNil: //LNil keep the same
		return LNil, true
	case LTUserData:
		if j, ok := v.(*LUserData).Value.(*Container); ok {
			return j.Data(), true
		} else {
			return nil, false
		}
	default:
		return nil, false
	}
}
func pack(v *Container, s *LState) LValue {
	switch t := v.Data().(type) {
	case LValue:
		return t
	case string:
		return LString(t)
	case bool:
		if t {
			return LTrue
		}
		return LFalse
	case types.Nil:
		return LNil
	case uint:
		return LNumber(t)
	case int:
		return LNumber(t)
	case int8:
		return LNumber(t)
	case int16:
		return LNumber(t)
	case int32:
		return LNumber(t)
	case int64:
		return LNumber(t)
	case uint8:
		return LNumber(t)
	case uint16:
		return LNumber(t)
	case uint32:
		return LNumber(t)
	case uint64:
		return LNumber(t)
	case float32:
		return LNumber(t)
	case float64:
		return LNumber(t)
	default:
		if m := v.ChildrenMap(); m != nil && len(m) > 0 {
			tx := s.NewTable()
			for key, container := range m {
				tx.RawSetString(key, pack(container, s))
			}
			return tx
		} else if arr := v.Children(); arr != nil {
			tx := s.NewTable()
			for _, container := range arr {
				tx.Append(pack(container, s))
			}
			return tx
		}
	}
	panic(fmt.Errorf("unsupported type: %T", v))
}
