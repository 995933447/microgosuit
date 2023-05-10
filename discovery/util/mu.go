package util

import (
	"github.com/gzjjyz/srvlib/lock"
)

type SpecSrvMuFactory struct {
	*lock.MulElemMuFactory
}

func NewSpecSrvMuFactory() *SpecSrvMuFactory {
	return &SpecSrvMuFactory{
		MulElemMuFactory: lock.NewMulElemMuFactory(),
	}
}

func (m *SpecSrvMuFactory) MakeOrGetOpOneSrvMu(srvName string) *lock.WithUsageMu {
	return m.MakeOrGetSpecElemMu(srvName)
}
