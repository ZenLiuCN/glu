package glu

import lua "github.com/yuin/gopher-lua"

/*
generic functions
*/
var (
	FmtErrMissing = "required value at %d"
	FmtErrType    = "required type not match at %d"
)

func Check[T any](s *lua.LState, n int, def T, cast func(v lua.LValue) (val T, ok bool)) (T, bool) {
	if s.GetTop() < n {
		s.RaiseError(FmtErrMissing, n)
		return def, false
	}
	a, ok := cast(s.Get(n))
	if !ok {
		s.RaiseError(FmtErrType, n)
		return def, false
	}
	return a, true
}

// CheckString return value and true only when value exists and is string.  Otherwise, an error raised.
func CheckString(s *lua.LState, n int) (string, bool) {
	return Check(s, n, "", func(v lua.LValue) (val string, ok bool) {
		if v.Type() == lua.LTString {
			return v.String(), true
		}
		return
	})
}

// CheckBool return value and true only when value exists and is bool. Otherwise, an error raised.
func CheckBool(s *lua.LState, n int) (bool, bool) {
	return Check(s, n, false, func(v lua.LValue) (val bool, ok bool) {
		if v.Type() == lua.LTBool {
			return v == lua.LTrue, true
		}
		return
	})
}

// CheckInt return value and true only when value exists and is exactly equals to the wanted number type. Returns converted number and false when value is number
func CheckInt(s *lua.LState, n int) (int, bool) {
	return Check(s, n, 0, func(v lua.LValue) (val int, ok bool) {
		if v.Type() == lua.LTNumber {
			f := float64(v.(lua.LNumber))
			i := int(f)
			return i, f == float64(i)
		}
		return
	})
}

// CheckInt16 return value and true only when value exists and is exactly equals to the wanted number type. Returns converted number and false when value is number
func CheckInt16(s *lua.LState, n int) (int16, bool) {
	return Check(s, n, 0, func(v lua.LValue) (val int16, ok bool) {
		if v.Type() == lua.LTNumber {
			f := float64(v.(lua.LNumber))
			i := int16(f)
			return i, f == float64(i)
		}
		return
	})
}

// CheckInt32 return value and true only when value exists and is exactly equals to the wanted number type. Returns converted number and false when value is number
func CheckInt32(s *lua.LState, n int) (int32, bool) {
	return Check(s, n, 0, func(v lua.LValue) (val int32, ok bool) {
		if v.Type() == lua.LTNumber {
			f := float64(v.(lua.LNumber))
			i := int32(f)
			return i, f == float64(i)
		}
		return
	})
}

// CheckInt64 return value and true only when value exists and is exactly equals to the wanted number type. Returns converted number and false when value is number
func CheckInt64(s *lua.LState, n int) (int64, bool) {
	return Check(s, n, 0, func(v lua.LValue) (val int64, ok bool) {
		if v.Type() == lua.LTNumber {
			f := float64(v.(lua.LNumber))
			i := int64(f)
			return i, f == float64(i)
		}
		return
	})
}

// CheckFloat32 return value and true only when value exists and is exactly equals to the wanted number type. Returns converted number and false when value is number
func CheckFloat32(s *lua.LState, n int) (float32, bool) {
	return Check(s, n, 0, func(v lua.LValue) (val float32, ok bool) {
		if v.Type() == lua.LTNumber {
			f := float64(v.(lua.LNumber))
			i := float32(f)
			return i, f == float64(i)
		}
		return
	})
}

// CheckFloat64 return value and true only when value exists and is exactly equals to the wanted number type.
func CheckFloat64(s *lua.LState, n int) (float64, bool) {
	return Check(s, n, 0, func(v lua.LValue) (val float64, ok bool) {
		if v.Type() == lua.LTNumber {
			return float64(v.(lua.LNumber)), true
		}
		return
	})
}

// CheckUserData return value and true only when value exists and can cast to the wanted type. Otherwise, an error raised.
func CheckUserData[T any](s *lua.LState, n int, def T, cast func(v any) (val T, ok bool)) (T, bool) {
	if s.GetTop() < n {
		s.RaiseError(FmtErrMissing, n)
		return def, false
	}
	v := s.Get(n)
	if v.Type() != lua.LTUserData {
		s.RaiseError(FmtErrType, n)
		return def, false
	}
	a, ok := cast(v.(*lua.LUserData).Value)
	if !ok {
		s.RaiseError(FmtErrType, n)
		return def, false
	}
	return a, true
}

// CheckRecUserData check the receiver as userdata of wanted type.
func CheckRecUserData[T any](s *lua.LState, ud *lua.LUserData, def T, cast func(v any) (val T, ok bool)) (T, bool) {
	if ud == nil {
		s.RaiseError(FmtErrMissing, 1)
		return def, false
	}
	a, ok := cast(ud.Value)
	if !ok {
		s.RaiseError(FmtErrType, 1)
		return def, false
	}
	return a, true
}

// Raise recover panic and raise error to Lua
func Raise(s *lua.LState, act func() int) (ret int) {
	defer func() {
		if r := recover(); r != nil {
			switch er := r.(type) {
			case error:
				s.RaiseError(er.Error())
			case string:
				s.RaiseError(`failure: %s`, er)
			default:
				s.RaiseError(`failure: %s`, er)
			}
			ret = 0
		}
	}()
	return act()
}
