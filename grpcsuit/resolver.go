package grpcsuit

import (
	"context"
	"fmt"
	"github.com/995933447/microgosuit/discovery"
	"github.com/995933447/microgosuit/log"
	"google.golang.org/grpc/resolver"
)

func NewResolver(srvName string, cc resolver.ClientConn, builder *Builder) *Resolver {
	return &Resolver{
		srvName: srvName,
		cc:      cc,
		Builder: builder,
	}
}

type Resolver struct {
	srvName string
	cc      resolver.ClientConn
	*Builder
}

func (r *Resolver) ResolveNow(options resolver.ResolveNowOptions) {
	srv, err := r.Builder.discover.Discover(context.Background(), r.srvName)
	if err != nil {
		log.Logger.Error(nil, err)
		return
	}
	r.UpdateSrvCfg(srv)
}

func (r *Resolver) Close() {
	r.Builder.OnResolverClosed(r)
}

func (r *Resolver) UpdateSrvCfg(srv *discovery.Service) {
	if srv.SrvName != r.srvName {
		return
	}

	state := resolver.State{}
	for _, node := range srv.Nodes {
		state.Addresses = append(state.Addresses, resolver.Address{
			Addr: fmt.Sprintf("%s:%d", node.Host, node.Port),
		})
	}
	state.ServiceConfig = r.cc.ParseServiceConfig(`{"loadBalancingPolicy":"round_robin"}`)

	r.cc.UpdateState(state)
}

var _ resolver.Resolver = (*Resolver)(nil)
