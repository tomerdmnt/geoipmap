package main

import (
	"github.com/layeh/gopher-luar"
	"github.com/yuin/gopher-lua"
)

func callScript(script string, record Record) (*Record, error) {
	L := lua.NewState()
	defer L.Close()

	if err := L.DoFile(script); err != nil {
		return nil, err
	}

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
