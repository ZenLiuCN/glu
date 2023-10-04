package glu

import (
	"errors"
	"fmt"
	. "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
	"reflect"
	"strings"
)

// CompileChunk compile code to FunctionProto
func CompileChunk(code string, source string) (*FunctionProto, error) {
	name := fmt.Sprintf(source)
	chunk, err := parse.Parse(strings.NewReader(code), name)
	if err != nil {
		return nil, err
	}
	return Compile(chunk, name)
}

// Operator operate stored state
type Operator = func(s *Vm) error

var (
	//OpNone operator do nothing, better to use nil
	OpNone = func(s *Vm) error { return nil }
	//OpPush operator to push N value
	OpPush = func(n ...LValue) Operator {
		return func(s *Vm) error {
			for _, value := range n {
				s.Push(value)
			}
			return nil
		}
	}
	//OpPushUserData operator to push N UserDate
	OpPushUserData = func(n ...any) Operator {
		return func(s *Vm) error {
			for _, i := range n {
				ud := s.NewUserData()
				ud.Value = i
				s.Push(ud)
			}
			return nil
		}
	}
	//OpSafe operator to wrap as none error will happen
	OpSafe = func(fn func(s *Vm)) Operator {
		return func(s *Vm) error {
			fn(s)
			return nil
		}
	}
	//OpPop operator to pop and consume N value from start,then pop to n
	OpPop = func(fn func(value ...LValue), start, count int) Operator {
		if start < 0 || count <= 0 {
			panic("start must greater than 0 and count must greater than 0")
		}
		t := start + count
		return func(s *Vm) error {
			v := make([]LValue, 0, count)
			for i := start; i < t; i++ {
				v = append(v, s.Get(i))
			}
			s.Pop(t - 1)
			fn(v...)
			return nil
		}
	}
	//OpPopN operator to pop n value (Reset stack)
	OpPopN = func(count int) Operator {
		if count < 1 {
			panic("count must greater than 0")
		}
		return func(s *Vm) error {
			s.Pop(count)
			return nil
		}
	}
)

// ExecuteChunk execute pre complied FunctionProto
func ExecuteChunk(code *FunctionProto, argN, retN int, before Operator, after Operator) (err error) {
	s := Get()
	defer Put(s)
	s.Push(s.NewFunctionFromProto(code))
	if before != nil {
		if err = before(s); err != nil {
			return err
		}
	}
	err = s.PCall(argN, retN, nil)
	if err != nil {
		return err
	}
	if after != nil {
		return after(s)
	}
	return nil
}

// ExecuteCode run code in LState, use before to push args, after to extract return value
func ExecuteCode(code string, argsN, retN int, before Operator, after Operator) error {
	s := Get()
	defer Put(s)
	if fn, err := s.LoadString(code); err != nil {
		return err
	} else {
		s.Push(fn)
		if before != nil {
			if err = before(s); err != nil {
				return err
			}
		}
		err = s.PCall(argsN, retN, nil)
		if err != nil {
			return err
		}
		if after != nil {
			return after(s)
		}
		return nil
	}
}

// TableToSlice convert LTable to a Slice with all Number index values
func TableToSlice(s *LTable) (r []LValue) {
	s.ForEach(func(key LValue, value LValue) {
		if key.Type() == LTNumber {
			r = append(r, value)
		}
	})
	return
}

// TableToMap convert LTable to a Map with all key values
func TableToMap(s *LTable) (r map[LValue]LValue) {
	r = make(map[LValue]LValue)
	s.ForEach(func(key LValue, value LValue) {
		r[key] = value
	})
	return
}

/*
TableUnpack convert LTable to a Map with all key values

All keys and values may be:

LTNumber: float64

LTBool: bool

LTTable: map[any]any

LTString: string

(those type will not output with noLua=true )

LTFunction: *LFunction

LTUserData: *LUserData

LChannel: LChannel
*/
func TableUnpack(s *LTable, noLua bool, history map[LValue]any) (r map[any]any, keys []any) {
	h := history
	if h == nil {
		h = make(map[LValue]any)
	}
	r = make(map[any]any)
	s.ForEach(func(key LValue, value LValue) {
		var k any
		var ok bool
		if k, ok = h[key]; !ok {
			switch key.Type() {
			case LTBool:
				k = key == LTrue
			case LTNumber:
				k = float64(key.(LNumber))
			case LTString:
				k = string(key.(LString))
			case LTFunction:
				if noLua {
					k = nil
				} else {
					k = key.(*LFunction)
				}
			case LTUserData:
				if noLua {
					k = nil
				} else {
					k = key.(*LUserData)
				}
			case LTThread:
				if noLua {
					k = nil
				} else {
					k = key.(*LState)
				}
			case LTTable:
				k, _ = TableUnpack(key.(*LTable), noLua, h)
			case LTChannel:
				if noLua {
					k = nil
				} else {
					k = value.(LChannel)
				}
			}
			h[key] = k
		}
		if k == nil {
			return
		}
		var val any
		keys = append(keys, k)
		if val, ok = h[value]; !ok {
			switch value.Type() {
			case LTBool:
				val = value == LTrue
			case LTNumber:
				val = float64(value.(LNumber))
			case LTString:
				val = string(value.(LString))
			case LTFunction:
				if noLua {
					val = nil
				} else {
					val = value.(*LFunction)
				}
			case LTUserData:
				if noLua {
					val = nil
				} else {
					val = value.(*LUserData)
				}
			case LTThread:
				if noLua {
					val = nil
				} else {
					val = value.(*LState)
				}
			case LTTable:
				val, _ = TableUnpack(value.(*LTable), noLua, h)
			case LTChannel:
				if noLua {
					val = nil
				} else {
					val = value.(LChannel)
				}
			}
			h[value] = val
		}
		if val != nil {
			r[k] = val
		}
	})
	return
}

var (
	//ErrorSuppress not raise error ,use for SafeFunc
	ErrorSuppress = errors.New("")
)

// Raw extract raw LValue: nil bool float64 string *LUserData *LState *LTable *LChannel
func Raw(v LValue) any {
	switch v.Type() {
	case LTNil:
		return nil
	case LTBool:
		return v == LTrue
	case LTNumber:
		return float64(v.(LNumber))
	case LTString:
		return v.String()
	case LTUserData:
		return v.(*LUserData)
	case LTThread:
		return v.(*LState)
	case LTTable:
		return v.(*LTable)
	case LTChannel:
		return v.(*LChannel)
	default:
		panic(fmt.Sprintf("unknown LuaType: %d", v.Type()))
	}
}

// Pack any to LValue.
//
// 1. nil, bool, numbers and other Lua value packed as normal LValue
//
// 2. array, slice,map[string]any packed into LTable (the elements also packed)
//
// 3. others are packed into LUserData
func Pack(v any, s *LState) LValue {
	switch v.(type) {
	case nil:
		return LNil
	case LValue:
		return v.(LValue)
	case bool:
		return LBool(v.(bool))
	case string:
		return LString(v.(string))
	case *LState:
		return v.(*LState)
	case *LFunction:
		return v.(*LFunction)
	case *LTable:
		return v.(*LTable)
	case *LUserData:
		return v.(*LUserData)
	case *LChannel:
		return v.(*LChannel)
	default:
		vv := reflect.ValueOf(v)
		switch {
		case vv.Kind() < reflect.Complex64:
			switch {
			case vv.CanInt():
				return LNumber(vv.Int())
			case vv.CanFloat():
				return LNumber(vv.Float())
			default:
				panic(fmt.Sprintf("unknown : %#v", v))
			}
		case vv.Kind() == reflect.Array || vv.Kind() == reflect.Slice:
			t := s.NewTable()
			for i := 0; i < vv.Len(); i++ {
				t.Insert(i, Pack(vv.Index(i).Interface(), s))
			}
			return t
		case vv.Kind() == reflect.Map:
			kt := vv.Type().Key()
			if (kt.Kind() > reflect.Bool && kt.Kind() < reflect.Complex64) || kt.Kind() == reflect.String {
				t := s.NewTable()
				r := vv.MapRange()
				for r.Next() {
					k := Pack(r.Key().Interface(), s)
					val := Pack(r.Value().Interface(), s)
					t.RawSet(k, val)
				}
				return t
			}
			u := s.NewUserData()
			u.Value = v
			return u
		default:
			u := s.NewUserData()
			u.Value = v
			return u
		}
	}

}
