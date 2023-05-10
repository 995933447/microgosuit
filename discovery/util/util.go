package util

import (
	"context"
	"github.com/995933447/microgosuit/discovery"
	"math/rand"
)

func Route(ctx context.Context, discover discovery.Discovery, srvName string) (*discovery.Node, error) {
	srv, err := discover.Discover(ctx, srvName)
	if err != nil {
		return nil, err
	}

	if len(srv.Nodes) == 0 {
		return nil, discovery.ErrNodeNotFound
	}

	nodeNum := len(srv.Nodes)
	randN := rand.Int()
	for i := 0; i < nodeNum; i++ {
		node := srv.Nodes[randN%nodeNum]
		if node.Available() {
			return node, nil
		}
		randN++
	}

	return nil, discovery.ErrNodeNotFound
}
