package glu

import (
	"errors"
	"fmt"
	. "github.com/yuin/gopher-lua"
	"testing"
)

var (
	c1 *FunctionProto
	c2 *FunctionProto
	c3 *FunctionProto
	c4 *FunctionProto
)

func init() {
	c1, _ = CompileChunk(`local a=1 print(a)`, `test1`)
	c2, _ = CompileChunk(`local a=... assert(a==1.1)`, `test2`)
	c3, _ = CompileChunk(`local a=... assert(a~=1.1)`, `test3`)
	c4, _ = CompileChunk(`local a=... assert(a~=1.1) return a`, `test4`)
}
func TestCompileChunk(t *testing.T) {
	_, err := CompileChunk(`local a=1 print(a)`, `test`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CompileChunk(`local`, `test`)
	if err == nil {
		t.Fatal(err)
	}
}

func TestExecuteChunk(t *testing.T) {
	type args struct {
		code   *FunctionProto
		argN   int
		retN   int
		before func(s *StoredState) error
		after  func(s *StoredState) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"c1", args{c1, 0, 0, nil, nil}, false},
		{"c2", args{c2, 1, 0, OpPush(LNumber(1.1)), nil}, false},
		{"c3", args{c3, 1, 0, OpPush(LNumber(1.1)), nil}, true},
		{"c4", args{c4, 1, 1, OpPush(LNumber(1.2)), func(s *StoredState) error {
			if s.Get(1).(LNumber) != 1.2 {
				return fmt.Errorf("just error")
			}
			return nil
		}}, false},
		{"c5", args{c4, 1, 1, func(s *StoredState) error {
			return fmt.Errorf("just error")
		}, func(s *StoredState) error {
			if s.CheckNumber(1) != 1.2 {
				return fmt.Errorf("just error")
			}
			return nil
		}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ExecuteChunk(tt.args.code, tt.args.argN, tt.args.retN, tt.args.before, tt.args.after); (err != nil) != tt.wantErr {
				t.Errorf("ExecuteChunk() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecuteCode(t *testing.T) {
	type args struct {
		code   string
		argsN  int
		retN   int
		before func(s *StoredState) error
		after  func(s *StoredState) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"simple", args{`local a=... print(a)`, 1, 0, func(s *StoredState) error {
			s.Push(LString("1"))
			return nil
		}, nil}, false},
		{"simple error", args{`local a=... assert(a==nil)`, 1, 0, func(s *StoredState) error {
			s.Push(LString("1"))
			return nil
		}, nil}, true},
		{"with return ", args{`local a=... return a`, 1, 1, func(s *StoredState) error {
			s.Push(LString("1"))
			return nil
		}, func(s *StoredState) error {
			if s.CheckString(1) != "1" {
				return fmt.Errorf("should '1'")
			}
			return nil
		}}, false},
		{"before error", args{`local a=... return a`, 1, 1, func(s *StoredState) error {
			return errors.New("just ")
		}, func(s *StoredState) error {
			if s.CheckString(1) != "1" {
				return fmt.Errorf("should '1'")
			}
			return nil
		}}, true},
		{"after error", args{`local a=... return a`, 1, 1, func(s *StoredState) error {
			s.Push(LNumber(1))
			return nil
		}, func(s *StoredState) error {
			return fmt.Errorf("should '1'")
		}}, true},
		{"invoke error", args{`local a=... error(tostring(a),1)`, 1, 1, func(s *StoredState) error {
			s.Push(LNumber(1))
			return nil
		}, func(s *StoredState) error {
			return fmt.Errorf("should '1'")
		}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ExecuteCode(tt.args.code, tt.args.argsN, tt.args.retN, tt.args.before, tt.args.after); (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOperator(t *testing.T) {
	s := Get()
	defer Put(s)
	err := OpNone(s)
	if err != nil {
		t.Fatal(err)
	}
	err = OpSafe(func(s *StoredState) {

	})(s)
	if err != nil {
		t.Fatal(err)
	}
	err = OpPushUserData(1)(s)
	if err != nil {
		return
	}
	if s.CheckUserData(1).Value.(int) != 1 {
		t.Fatal()
	}
	s.Pop(1)
	err = OpPush(LNumber(1), LNumber(2), LNumber(3))(s)
	if err != nil {
		return
	}
	for i := 1; i < 4; i++ {
		if s.CheckInt(i) != i {
			t.Fatal()
		}
	}
	s.Pop(3)
	err = OpPush(LNumber(1), LNumber(2), LNumber(3))(s)
	if err != nil {
		return
	}
	err = OpPop(func(v ...LValue) {
		if v[0].String() != "1" ||
			v[1].String() != "2" ||
			v[2].String() != "3" {
			t.Fail()
		}
	}, 1, 3)(s)
	if err != nil {
		return
	}
}
