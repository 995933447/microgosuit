package microgosuit

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/995933447/gonetutil"
	"github.com/995933447/microgosuit/discovery"
	"github.com/995933447/microgosuit/env"
	"github.com/995933447/microgosuit/factory"
	"github.com/995933447/microgosuit/grpcsuit"
	"github.com/995933447/microgosuit/grpcsuit/handler/health"
	"github.com/995933447/microgosuit/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func InitSuitWithGrpc(ctx context.Context, metaFilePath, resolveSchema, discoverPrefix string) error {
	if err := env.InitMeta(metaFilePath); err != nil {
		return err
	}

	if err := grpcsuit.InitGrpcResolver(ctx, resolveSchema, discoverPrefix); err != nil {
		return err
	}

	return nil
}

type ServeGrpcReq struct {
	RegDiscoverKeyPrefix            string
	SrvName                         string
	IpVar                           string
	Port                            int
	PProfIpVar                      string
	PProfPort                       int
	RegisterCustomServiceServerFunc func(*grpc.Server) error
	BeforeRegDiscover               func(discovery.Discovery, *discovery.Node) error
	AfterRegDiscover                func(discovery.Discovery, *discovery.Node) error
	OnReady                         func(*grpc.Server)
	EnabledHealth                   bool
	SrvOpts                         []grpc.ServerOption
}

func ServeGrpc(ctx context.Context, req *ServeGrpcReq) error {
	if req.PProfIpVar != "" && req.PProfPort > 0 {
		go func() {
			ip, err := gonetutil.EvalVarToParseIp(req.PProfIpVar)
			if err != nil {
				log.Logger.Error(nil, err)
				return
			}

			err = http.ListenAndServe(fmt.Sprintf("%s:%d", ip, req.PProfPort), nil)
			if err != nil {
				log.Logger.Error(nil, err)
			}
		}()
	}

	ip, err := gonetutil.EvalVarToParseIp(req.IpVar)
	if err != nil {
		return err
	}

	node := discovery.NewNode(ip, req.Port)
	grpcServer := grpc.NewServer(req.SrvOpts...)
	if req.RegisterCustomServiceServerFunc != nil {
		if err = req.RegisterCustomServiceServerFunc(grpcServer); err != nil {
			return err
		}
	}

	reflection.Register(grpcServer)

	if req.EnabledHealth {
		health.RegisterHealthReporterServer(grpcServer, &health.Reporter{})
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, req.Port))
	if err != nil {
		return err
	}

	discover, err := factory.GetOrMakeDiscovery(req.RegDiscoverKeyPrefix)
	if err != nil {
		return err
	}

	if req.BeforeRegDiscover != nil {
		if err = req.BeforeRegDiscover(discover, node); err != nil {
			return err
		}
	}

	err = discover.Register(ctx, req.SrvName, node)
	if err != nil {
		return err
	}

	if req.AfterRegDiscover != nil {
		if err = req.AfterRegDiscover(discover, node); err != nil {
			return err
		}
	}

	defer func() {
		err = discover.Unregister(ctx, req.SrvName, node, true)
		if err != nil {
			log.Logger.Error(ctx, err)
		}
	}()

	if req.OnReady != nil {
		req.OnReady(grpcServer)
	}

	err = grpcServer.Serve(listener)
	if err != nil {
		return err
	}

	return nil
}
