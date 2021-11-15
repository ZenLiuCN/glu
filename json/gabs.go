package json

import (
	. "github.com/Jeffail/gabs/v2"
	. "github.com/yuin/gopher-lua"
	"glu"
)

var (
	JsonType      *glu.Type
	JsonModule    *glu.Module
	JsonTypeCheck = func(l *LState, n int) *Container {
		ud := l.CheckUserData(n)
		if v, ok := ud.Value.(*Container); ok {
			return v
		}
		l.ArgError(1, "json.Json expected")
		return nil
	}
)

func init() {
	check := func(l *LState) *Container {
		return JsonTypeCheck(l, 1)
	}
	m := glu.NewModular("json", `module json is wrap of jeffail/gabs as dynamic json tool.
json.Json  ==> type, json container.
`, true)
	t := glu.NewType("Json", false, `Json.new(json string?)Json? ==> create Json instance.`,
		func(s *LState) interface{} {
			if s.GetTop() == 1 {
				s.CheckType(1, LTString)
				v, err := ParseJSON([]byte(s.ToString(1)))
				if err != nil {
					s.ArgError(1, "invalid JSON string")
					return 0
				}
				return v
			}
			return New()
		})
	t.
		AddMethod("json", `Json:json(pretty boolean=false,ident string='\t')string ==> json string of the Json, if empty will be '{}'.`,
			func(s *LState) int {
				v := check(s)
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
		AddMethod("path", `Json:path(path string)Json?  ==> fetch Json by path, path is gabs path.`,
			func(s *LState) int {
				v := check(s)
				s.CheckType(2, LTString)
				p := s.ToString(2)
				if v.ExistsP(p) {
					return t.New(s, v.Path(p))
				}
				s.Push(LNil)
				return 1
			}).
		/*		AddMethod("pointer", `Json:pointer(pointer string)Json?  ==> fetch Json by pointer, pointer is json pointer [rfc6901].`,
				func(s *LState) int {
					v := check(s)
					s.CheckType(2, LTString)
					p := s.ToString(2)
					x, err := v.JSONPointer(p)
					if err != nil {
						s.RaiseError(" Json:pointer error %s", err)
						return 0
					}
					return t.New(s, x)
				}).*/
		AddMethod("exists", `Json:exists(path string)boolean  ==> check existence of path.`,
			func(s *LState) int {
				v := check(s)
				s.CheckType(2, LTString)
				p := s.ToString(2)
				if v.ExistsP(p) {
					s.Push(LTrue)
				} else {
					s.Push(LFalse)
				}
				return 1
			}).
		AddMethod("at", `Json:at(key int|string)Json?  ==> fetch value at index for array or key for object.`,
			func(s *LState) int {
				v := check(s)
				s.CheckTypes(2, LTString, LTNumber)
				p := s.Get(2)
				if p.Type() == LTString {
					x := p.String()
					if v.ExistsP(x) {
						return t.New(s, v.Path(x))
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
					return t.New(s, x)
				}
				return 0
			}).
		AddMethod("type", `Json:type()int  ==> fetch JSON type: 0 nil,1 string,2 number,3 boolean,4 array,5 object.`,
			func(s *LState) int {
				v := check(s)
				if _, ok := v.Data().(string); ok {
					s.Push(LNumber(1))
				} else if _, ok = v.Data().(float64); ok {
					s.Push(LNumber(2))
				} else if _, ok = v.Data().(bool); ok {
					s.Push(LNumber(3))
				} else if _, ok = v.Data().(interface{}); ok {
					s.Push(LNumber(4))
				} else if _, ok = v.Data().(map[string]interface{}); ok {
					s.Push(LNumber(5))
				} else {
					s.Push(LNumber(0))
				}
				return 1
			}).
		AddMethod("set", `Json:set(path string, json JSON|string|number|bool|nil)  ==> set value at path.if value is nil,will delete it.this can't append array.'`,
			func(s *LState) int {
				v := check(s)
				s.CheckType(2, LTString)
				s.CheckTypes(3, LTUserData, LTString, LTNumber, LTBool, LTNil)
				p := s.ToString(2)
				x := s.Get(3)
				switch x.Type() {
				case LTUserData:
					if j, ok := x.(*LUserData).Value.(*Container); ok {
						_, err := v.SetP(j, p)
						if err != nil {
							s.RaiseError("set value error: %s", err)
							return 0
						}
					} else {
						s.ArgError(3, "need JSON")
					}
				case LTString:
					_, err := v.SetP(s.ToString(3), p)
					if err != nil {
						s.RaiseError("set value error: %s", err)
						return 0
					}
				case LTNumber:
					_, err := v.SetP(float64(s.ToNumber(3)), p)
					if err != nil {
						s.RaiseError("set value error: %s", err)
						return 0
					}
				case LTBool:
					_, err := v.SetP(s.ToBool(3), p)
					if err != nil {
						s.RaiseError("set value error: %s", err)
						return 0
					}
				case LTNil:
					if err := v.DeleteP(p); err != nil {
						s.RaiseError("set value error: %s", err)
						return 0
					}
				default:
					s.ArgError(3, "invalid type")
				}
				return 0
			}).
		AddMethod("append", `Json:append(path string, json JSON|string|number|bool|nil)  ==> append value, path must pointer to array.`,
			func(s *LState) int {
				v := check(s)
				s.CheckType(2, LTString)
				s.CheckTypes(3, LTUserData, LTString, LTNumber, LTBool, LTNil)
				p := s.ToString(2)
				v0 := v.Path(p)
				if v0 == nil {
					s.RaiseError("value at path is object")
					return 0
				} else if _, ok := v0.Data().(map[string]interface{}); ok {
					s.RaiseError("value at path is object")
					return 0
				}
				x := s.Get(3)
				switch x.Type() {
				case LTUserData:
					if j, ok := x.(*LUserData).Value.(*Container); ok {
						err := v.ArrayAppendP(j, p)
						if err != nil {
							s.RaiseError("append value error: %s", err)
							return 0
						}
					} else {
						s.ArgError(3, "need JSON")
					}
				case LTString:
					if err := v.ArrayAppendP(s.ToString(3), p); err != nil {
						s.RaiseError("append value error: %s", err)
						return 0
					}

				case LTNumber:
					if err := v.ArrayAppendP(float64(s.ToNumber(3)), p); err != nil {
						s.RaiseError("append value error: %s", err)
						return 0
					}
				case LTBool:
					if err := v.ArrayAppendP(s.ToBool(3), p); err != nil {
						s.RaiseError("append value error: %s", err)
						return 0
					}
				case LTNil:
					if err := v.DeleteP(p); err != nil {
						s.RaiseError("set value error: %s", err)
						return 0
					}
				default:
					s.ArgError(3, "invalid type")
				}
				return 0
			}).
		AddMethod("array", `Json:array(path string)bool  ==> check if it's array at path.`,
			func(s *LState) int {
				v := check(s)
				s.CheckType(2, LTString)
				p := s.ToString(2)
				if !v.ExistsP(p) {
					s.Push(LFalse)
				} else {
					if _, ok := v.Path(p).Data().([]interface{}); ok {
						s.Push(LTrue)
					} else {
						s.Push(LFalse)
					}
				}
				return 1
			}).
		AddMethod("object", `Json:object(path string)bool  ==> check if it's object at path.`,
			func(s *LState) int {
				v := check(s)
				s.CheckType(2, LTString)
				p := s.ToString(2)
				if !v.ExistsP(p) {
					s.Push(LFalse)
				} else {
					if _, ok := v.Path(p).Data().(map[string]interface{}); ok {
						s.Push(LTrue)
					} else {
						s.Push(LFalse)
					}
				}
				return 1
			}).
		AddMethod("asBool", `Json:asBool(path string)bool  ==> fetch value as boolean, if not exists return false.`,
			func(s *LState) int {
				v := check(s)
				s.CheckType(2, LTString)
				p := s.ToString(2)
				if !v.ExistsP(p) {
					s.Push(LFalse)
				} else if b, ok := v.Path(p).Data().(bool); !ok {
					s.RaiseError("value not bool")
					return 0
				} else if b {
					s.Push(LTrue)
				} else {
					s.Push(LFalse)
				}
				return 1
			}).
		AddMethod("asString", `Json:asString(path string)string  ==> fetch value as string, if not exists return nil.`,
			func(s *LState) int {
				v := check(s)
				s.CheckType(2, LTString)
				p := s.ToString(2)
				if !v.ExistsP(p) {
					s.Push(LNil)
				} else if b, ok := v.Path(p).Data().(string); !ok {
					s.RaiseError("value not string")
					return 0
				} else {
					s.Push(LString(b))
				}
				return 1
			}).
		AddMethod("asNumber", `Json:asNumber(path string)number  ==> fetch value as number, if not exists return nil.`,
			func(s *LState) int {
				v := check(s)
				s.CheckType(2, LTString)
				p := s.ToString(2)
				if !v.ExistsP(p) {
					s.Push(LNil)
				} else if b, ok := v.Path(p).Data().(float64); !ok {
					s.RaiseError("value not string")
					return 0
				} else {
					s.Push(LNumber(b))
				}
				return 1
			})

	m.AddFunc("from", `json.from(val table|number|string|boolean)Json ==> create json from value`,
		func(s *LState) int {
			s.CheckTypes(1, LTString, LTNumber, LTBool, LTTable)
			v := s.Get(1)
			switch v.Type() {
			case LTString:
				g := New()
				_, _ = g.Set(s.ToString(1))
				return t.New(s, g)
			case LTNumber:
				g := New()
				_, _ = g.Set(float64(s.ToNumber(1)))
				return t.New(s, g)
			case LTBool:
				g := New()
				_, _ = g.Set(s.ToBool(1))
				return t.New(s, g)
			case LTTable:
				return t.New(s, parseTable(s.ToTable(1), New()))
			default:
				s.ArgError(1, "invalid")
				return 0
			}
		})
	m.AddModule(t)
	JsonType = t
	JsonModule = m
	glu.Registry = append(glu.Registry, JsonModule)

}
func parseTable(t *LTable, g *Container) *Container {
	arr := t.MaxN() == t.Len()
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
