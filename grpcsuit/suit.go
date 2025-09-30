package grpcsuit

import (
	"context"
	"fmt"
	"sync"

	"github.com/995933447/microgosuit/discovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

var RoundRobinDialOpts = []grpc.DialOption{
	grpc.WithTransportCredentials(insecure.NewCredentials()),
	grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingConfig": [{"%s":{}}]}`, roundrobin.Name)),
}

var NotRoundRobinDialOpts = []grpc.DialOption{
	grpc.WithTransportCredentials(insecure.NewCredentials()),
}

var (
	customizedDialOpts                  []grpc.DialOption
	customizedDialOptsMergedDefault     []grpc.DialOption
	customizedDialOptsMergedDefaultOnce sync.Once
)

func RegisterCustomizedDialOpts(opts ...grpc.DialOption) {
	customizedDialOpts = append(customizedDialOpts, opts...)
	customizedDialOptsMergedDefault = nil
}

func GetCustomizedOpts() []grpc.DialOption {
	return customizedDialOpts
}

func GetCustomizedOptsDefault() []grpc.DialOption {
	if len(customizedDialOpts) == 0 {
		return RoundRobinDialOpts
	}
	return customizedDialOpts
}

func GetCustomizedOptsMergedDefault() []grpc.DialOption {
	if len(customizedDialOptsMergedDefault) == 0 {
		customizedDialOptsMergedDefaultOnce.Do(func() {
			customizedDialOptsMergedDefault = make([]grpc.DialOption, len(RoundRobinDialOpts))
			copy(customizedDialOptsMergedDefault, RoundRobinDialOpts)
			customizedDialOptsMergedDefault = append(customizedDialOptsMergedDefault, customizedDialOpts...)
		})
	}
	return customizedDialOptsMergedDefault
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
