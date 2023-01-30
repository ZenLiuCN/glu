package json

import (
	. "github.com/Jeffail/gabs/v2"
	. "github.com/ZenLiuCN/glu"
	. "github.com/yuin/gopher-lua"
)

var (
	JsonType   Type
	JsonModule Module
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
	JsonModule = NewModule("json", `module json is wrap of jeffail/gabs as dynamic json tool.
json.Json  ==> type, json container.
`, true)
	JsonType = NewTypeCast(new(Container), "Json", `Json.new(json string?)Json? ==> create Json instance.`, false,
		func(s *LState) interface{} {
			if s.GetTop() == 1 {
				v, err := ParseJSON([]byte(s.CheckString(1)))
				if err != nil {
					s.ArgError(1, "invalid JSON string")
					return 0
				}
				return v
			}
			return New()
		}).
		AddMethodCast("json", `Json:json(pretty boolean=false,ident string='\t')string ==> json string of the Json, if empty will be '{}'.`,
			func(s *LState, data interface{}) int {
				v := data.(*Container)
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
		AddMethodCast("path", `Json:path(path string?)Json?  ==> fetch Json by path, path is gabs path.`,
			func(s *LState, data interface{}) int {
				v := data.(*Container)
				p := s.CheckString(2)
				if v.ExistsP(p) {
					return JsonType.New(s, v.Path(p))
				}
				s.Push(LNil)
				return 1
			}).
		AddMethodCast("exists", `Json:exists(path string?)boolean  ==> check existence of path.`,
			func(s *LState, data interface{}) int {
				v := data.(*Container)
				p, _ := checkPath(s, v, 0)
				s.Push(LBool(p != nil))
				return 1
			}).
		AddMethodCast("at", `Json:at(key int|string)Json?  ==> fetch value at index for array or key for object.`,
			func(s *LState, data interface{}) int {
				v := data.(*Container)
				s.CheckTypes(2, LTString, LTNumber)
				p := s.Get(2)
				if p.Type() == LTString {
					x := p.String()
					if v.ExistsP(x) {
						return JsonType.New(s, v.Path(x))
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
					return JsonType.New(s, x)
				}
				return 0
			},
		).
		AddMethodCast("type", `Json:type()int  ==> fetch JSON type: 0 nil,1 string,2 number,3 boolean,4 array,5 object.`,
			func(s *LState, data interface{}) int {
				v := data.(*Container)
				if _, ok := v.Data().(string); ok {
					s.Push(LNumber(1))
				} else if _, ok = v.Data().(float64); ok {
					s.Push(LNumber(2))
				} else if _, ok = v.Data().(bool); ok {
					s.Push(LNumber(3))
				} else if _, ok = v.Data().([]interface{}); ok {
					s.Push(LNumber(4))
				} else if _, ok = v.Data().(map[string]interface{}); ok {
					s.Push(LNumber(5))
				} else {
					s.Push(LNumber(0))
				}
				return 1
			}).
		AddMethodCast("set", `Json:set(path string?, json JSON|string|number|bool|nil)string?  ==> set value at path.if value is nil,will delete it.this can't append array.'`,
			func(s *LState, data interface{}) int {
				v := data.(*Container)
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
				val := unpack(x)
				var err error
				if val == nil {
					s.ArgError(idx, "invalid data type")
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
		AddMethodCast("append", `Json:append(path string?, json JSON|string|number|bool|nil)string?  ==> append value, path must pointer to array.`,
			func(s *LState, data interface{}) int {
				c := data.(*Container)
				_, p := checkPath(s, c, 1)
				var idx int
				if p != "" {
					idx = 3
				} else {
					idx = 2
				}
				s.CheckTypes(idx, LTUserData, LTString, LTNumber, LTBool, LTNil)
				x := s.Get(idx)
				val := unpack(x)
				if val == nil {
					s.ArgError(idx, "invalid data type")
					return 0
				} else if val == LNil {
					val = nil
				}
				var err error
				if p == "" {
					if m, ok := c.Data().(map[string]interface{}); ok && len(m) > 0 {
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
						idx = len(c.Data().([]interface{})) - 1
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
				} else if m, ok := c.Path(p).Data().(map[string]interface{}); ok && len(m) > 0 {
					s.Push(LString("value at path is object"))
					return 1
				} else if ok {
					_, err = c.Set([]interface{}{}, p)
					if err != nil {
						s.Push(LString("append value fail:" + err.Error()))
						return 1
					}
				}
				if val == nil {
					idx = len(c.Path(p).Data().([]interface{})) - 1
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
		AddMethodCast("isArray", `Json:isArray(path string?)bool  ==> check if it's array at path.`,
			func(s *LState, data interface{}) int {
				c := data.(*Container)
				v, _ := checkPath(s, c, 0)
				if v == nil {
					s.Push(LFalse)
				} else {
					if _, ok := v.Data().([]interface{}); ok {
						s.Push(LTrue)
					} else {
						s.Push(LFalse)
					}
				}
				return 1
			}).
		AddMethodCast("isObject", `Json:isObject(path string?)bool  ==> check if it's object at path.`,
			func(s *LState, data interface{}) int {
				c := data.(*Container)
				v, _ := checkPath(s, c, 0)
				if v == nil {
					s.Push(LFalse)
				} else if _, ok := v.Data().(map[string]interface{}); ok {
					s.Push(LTrue)
				} else {
					s.Push(LFalse)
				}

				return 1
			}).
		AddMethodCast("bool", `Json:bool(path string?)bool  ==> fetch value as boolean, if not exists return false.`,
			func(s *LState, data interface{}) int {
				c := data.(*Container)
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
		AddMethodCast("string", `Json:string(path string?)string?  ==> fetch value as string, if not exists or not string return nil.`,
			func(s *LState, data interface{}) int {
				c := data.(*Container)
				v, _ := checkPath(s, c, 0)
				if v == nil {
					s.Push(LNil)
				} else if b, ok := v.Data().(string); !ok {
					s.Push(LNil)
				} else {
					s.Push(LString(b))
				}
				return 1
			}).
		AddMethodCast("number", `Json:number(path string?)number?  ==> fetch value as number, if not exists return nil.`,
			func(s *LState, data interface{}) int {
				c := data.(*Container)
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
		AddMethodCast("size", `Json:size(path string?)number?  ==> fetch  object size or array size else nil.`,
			func(s *LState, data interface{}) int {
				c := data.(*Container)
				v, _ := checkPath(s, c, 0)
				if b, ok := v.Data().(map[string]interface{}); ok {
					s.Push(LNumber(len(b)))
				} else if a, ok := v.Data().([]interface{}); ok {
					s.Push(LNumber(len(a)))
				} else if v == nil {
					s.Push(LNil)
				} else {
					s.Push(LNumber(1))
				}
				return 1
			})

	JsonModule.
		AddFunc("from", `json.from(val table|number|string|boolean)Json ==> create json from value`,
			func(s *LState) int {
				s.CheckTypes(1, LTString, LTNumber, LTBool, LTTable)
				v := s.Get(1)
				switch v.Type() {
				case LTString:
					g := New()
					_, _ = g.Set(s.ToString(1))
					return JsonType.New(s, g)
				case LTNumber:
					g := New()
					_, _ = g.Set(float64(s.ToNumber(1)))
					return JsonType.New(s, g)
				case LTBool:
					g := New()
					_, _ = g.Set(s.ToBool(1))
					return JsonType.New(s, g)
				case LTTable:
					return JsonType.New(s, parseTable(s.ToTable(1), New()))
				default:
					s.ArgError(1, "invalid")
					return 0
				}
			}).
		AddModule(JsonType)

	Success(Register(JsonModule))

}
func parseTable(t *LTable, g *Container) *Container {
	arr := t.MaxN() != 0 && t.MaxN() == t.Len()
	if arr {
		if _, ok := g.Data().(map[string]interface{}); ok {
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
			parseTable(v.(*LTable), o)
			if arr {
				_ = g.ArrayAppend(o)
			} else {
				_, _ = g.Set(o, k.String())
			}
		}
	})
	return g
}
func unpack(v LValue) interface{} {
	switch v.Type() {
	case LTString:
		return v.String()
	case LTNumber:
		return float64(v.(LNumber))
	case LTBool:
		return v == LTrue
	case LTNil:
		return LNil
	case LTUserData:
		if j, ok := v.(*LUserData).Value.(*Container); ok {
			return j.Data()
		} else {
			return nil
		}
	default:
		return nil
	}
}
