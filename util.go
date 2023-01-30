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
type Operator = func(s *StoredState) error

var (
	//OpNone operator do nothing, better to use nil
	OpNone = func(s *StoredState) error { return nil }
	//OpPush operator to push N value
	OpPush = func(n ...LValue) Operator {
		return func(s *StoredState) error {
			for _, value := range n {
				s.Push(value)
			}
			return nil
		}
	}
	//OpPushUserData operator to push N UserDate
	OpPushUserData = func(n ...interface{}) Operator {
		return func(s *StoredState) error {
			for _, i := range n {
				ud := s.NewUserData()
				ud.Value = i
				s.Push(ud)
			}
			return nil
		}
	}
	//OpSafe operator to wrap as none error will happen
	OpSafe = func(fn func(s *StoredState)) Operator {
		return func(s *StoredState) error {
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
		return func(s *StoredState) error {
			v := make([]LValue, 0, count)
			for i := start; i < t; i++ {
				v = append(v, s.Get(i))
			}
			s.Pop(t - 1)
			fn(v...)
			return nil
		}
	}
	//OpPopN operator to pop n value (restore stack)
	OpPopN = func(count int) Operator {
		if count < 1 {
			panic("count must greater than 0")
		}
		return func(s *StoredState) error {
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

// PanicErr check if error not nil then panic
func PanicErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Recoverable warp a function with recover
func Recoverable(act func()) (err error) {
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

func Recoverable2(act func() error) (err error) {
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
