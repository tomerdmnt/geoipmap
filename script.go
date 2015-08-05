package main

import (
	"github.com/layeh/gopher-luar"
	"github.com/yuin/gopher-lua"
)

func newLuaState(script string) (*lua.LState, error) {
	if script == "" {
		return nil, nil
	}
	L := lua.NewState()
	if err := L.DoFile(script); err != nil {
		return nil, err
	}
	L.SetGlobal("globals", L.NewTable())
	return L, nil
}

func callScript(L *lua.LState, record Record) (*Record, error) {
	lrecord := luar.New(L, &record)

	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("record"),
		NRet:    1,
		Protect: true,
	}, lrecord); err != nil {
		return nil, err
	}
	if result := L.Get(-1); result != lua.LFalse {
		return &record, nil
	} else {
		return nil, nil
	}
}
