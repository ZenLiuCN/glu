package glu

import (
	"fmt"
	. "github.com/yuin/gopher-lua"
	"testing"
)

func TestGluModule(t *testing.T) {
	err := ExecuteCode(`
		print(Help())
		print(Help('?'))
		print(Help('chunk'))
		local ch,err=chunk([[
			local c=...
			print(c)
			return c
		]],'testChunk')
		if err~=nil then error(err,1) end
		return ch
	`, 0, 1, nil, func(s *Vm) error {
		c := GluMod.CheckChunk(s.LState, 1)
		return ExecuteChunk(c, 1, 1, OpPush(LString("1")), func(s *Vm) error {
			if s.CheckString(1) != "1" {
				return fmt.Errorf("error: %s", s.CheckString(1))
			}
			return nil
		})
	})
	if err != nil {
		t.Fatal(err)
	}
}
