package health

import (
	"context"
)

type Reporter struct {
}

func (h *Reporter) Ping(_ context.Context, _ *PingReq) (*PingResp, error) {
	return &PingResp{}, nil

}

var _ HealthReporterServer = (*Reporter)(nil)
