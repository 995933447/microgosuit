package grpcsuit

import (
	"context"
	"fmt"
	"github.com/995933447/microgosuit/discovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/resolver"
)

var RoundRobinDialOpts = []grpc.DialOption{
	grpc.WithInsecure(),
	grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingConfig": [{"%s":{}}]}`, roundrobin.Name)),
}

var NotRoundRobinDialOpts = []grpc.DialOption{
	grpc.WithInsecure(),
}

var customDoOnDiscoverSrvUpdated discovery.OnSrvUpdatedFunc = func(ctx context.Context, evt discovery.Evt, srv *discovery.Service) {}

func InitGrpcResolver(ctx context.Context, resolveSchema, discoverPrefix string) error {
	builder, err := NewBuilder(ctx, resolveSchema, discoverPrefix)
	if err != nil {
		return err
	}
	resolver.Register(builder)
	return nil
}

func OnDiscoverSrvUpdated(fn discovery.OnSrvUpdatedFunc) {
	customDoOnDiscoverSrvUpdated = fn
}
