package health

import (
	"context"
)

type Reporter struct {
	UnimplementedHealthReporterServer
	serviceNames map[string]struct{}
}

func NewReporter(serviceNames []string) *Reporter {
	r := &Reporter{
		serviceNames: make(map[string]struct{}),
	}
	for _, serviceName := range serviceNames {
		r.serviceNames[serviceName] = struct{}{}
	}
	return r
}

func (r *Reporter) Ping(_ context.Context, req *PingReq) (*PingResp, error) {
	var resp PingResp
	if r.serviceNames != nil {
		_, resp.Ok = r.serviceNames[req.PingService]
	}
	return &resp, nil
}

var _ HealthReporterServer = (*Reporter)(nil)
