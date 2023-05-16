package util

import (
	"github.com/995933447/runtimeutil"
)

type SpecSrvMuFactory struct {
	*runtimeutil.MulElemMuFactory
}

func NewSpecSrvMuFactory() *SpecSrvMuFactory {
	return &SpecSrvMuFactory{
		MulElemMuFactory: runtimeutil.NewMulElemMuFactory(),
	}
}

func (m *SpecSrvMuFactory) MakeOrGetOpOneSrvMu(srvName string) *runtimeutil.WithUsageMu {
	return m.MakeOrGetSpecElemMu(srvName)
}
