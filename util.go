package glu

import (
	"errors"
	"fmt"
	. "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
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
	OpPushUserData = func(n ...interface{}) Operator {
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

//TableToSlice convert LTable to a Slice with all Number index values
func TableToSlice(s *LTable) (r []LValue) {
	s.ForEach(func(key LValue, value LValue) {
		if key.Type() == LTNumber {
			r = append(r, value)
		}
	})
	return
}

//TableToMap convert LTable to a Map with all key values
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
func TableUnpack(s *LTable, noLua bool, history map[LValue]interface{}) (r map[interface{}]interface{}, keys []interface{}) {
	h := history
	if h == nil {
		h = make(map[LValue]interface{})
	}
	r = make(map[interface{}]interface{})
	s.ForEach(func(key LValue, value LValue) {
		var k interface{}
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
		var val interface{}
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

// Recover warp a callable func with recover
func Recover(act func()) (err error) {
	defer func() {
		r := recover()
		switch r.(type) {
		case error:
			err = r.(error)
		case string:
			err = errors.New(r.(string))
		default:
			err = fmt.Errorf(`%#v`, r)
		}
	}()
	act()
	return
}

// RecoverErr   warp an error supplier func with recover
func RecoverErr(act func() error) (err error) {
	defer func() {
		r := recover()
		switch r.(type) {
		case error:
			err = r.(error)
		case string:
			err = errors.New(r.(string))
		default:
			err = fmt.Errorf(`%#v`, r)
		}
	}()
	err = act()
	return
}

// Success must have no error or-else throw
func Success(err error) {
	if err != nil {
		panic(err)
	}
}

// Failed must have error or-else throw
func Failed(err error) {
	if err == nil {
		panic("should fail")
	}
}
