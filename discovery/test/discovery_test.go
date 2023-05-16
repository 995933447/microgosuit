package test

import (
	"context"
	"fmt"
	"github.com/995933447/microgosuit/discovery"
	"github.com/995933447/microgosuit/discovery/impl/etcd"
	"github.com/995933447/microgosuit/discovery/impl/filecachedproxy"
	clientv3 "go.etcd.io/etcd/client/v3"
	"testing"
	"time"
)

func TestFileCacheDiscovery(t *testing.T) {
	conn, err := etcd.NewDiscovery("", time.Hour, clientv3.Config{
		Endpoints: []string{
			"127.0.0.1:12379",
		},
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	discover := filecachedproxy.NewDiscovery("D:\\dispatch", conn)
	discover.OnSrvUpdated(func(ctx context.Context, evt discovery.Evt, srv *discovery.Service) {
		t.Logf("evt:%d srvName:%s", evt, srv.SrvName)
		if evt != discovery.EvtDeleted {
			for _, cfg := range srv.Nodes {
				t.Logf("node:%+v", cfg)
			}
		}
		t.Logf("finish evt:%d", evt)
	})
	service, err := discover.Discover(context.Background(), "logv4")
	if err != nil {
		t.Fatal(err)
		return
	}

	t.Log(service)

	err = discover.Register(context.Background(), "logv3", &discovery.Node{
		Host: "127.2.1.1",
		Port: 120104,
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	err = discover.Register(context.Background(), "logv3", &discovery.Node{
		Host: "127.2.1.1",
		Port: 120103,
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	service, err = discover.Discover(context.Background(), "logv6")
	if err != nil && err != discovery.ErrSrvNotFound {
		t.Fatal(err)
		return
	}

	err = discover.Register(context.Background(), "logv6", &discovery.Node{
		Host: "127.0.3.2",
		Port: 120104,
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	service, err = discover.Discover(context.Background(), "logv6")
	if err != nil && err != discovery.ErrSrvNotFound {
		t.Fatal(err)
		return
	}

	if err = discover.Unregister(context.Background(), "logv6", service.Nodes[0], true); err != nil {
		t.Fatal(err)
	}

	//for _, node := range service.Nodes {
	//	if err = discover.Unregister(context.Background(), "logv6", node, true); err != nil {
	//		t.Fatal(err)
	//	}
	//}

	time.Sleep(time.Minute)
}

func TestEtcdDiscovery(t *testing.T) {
	discover, err := etcd.NewDiscovery("", time.Hour, clientv3.Config{
		Endpoints: []string{
			"127.0.0.1:12379",
		},
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	srv, err := discover.Discover(context.Background(), "logv3")
	if err != nil && err != discovery.ErrSrvNotFound {
		t.Fatal(err)
		return
	}

	if err == nil {
		t.Logf("srvName:%s", srv.SrvName)
		for _, cfg := range srv.Nodes {
			t.Logf("node:%+v", cfg)
		}
	}

	discover.OnSrvUpdated(func(ctx context.Context, evt discovery.Evt, srv *discovery.Service) {
		if evt != discovery.EvtDeleted {
			t.Logf("evt:%d srvName:%s", evt, srv.SrvName)
			for _, cfg := range srv.Nodes {
				t.Logf("node:%+v", cfg)
			}
		}
		t.Logf("finish evt:%d", evt)
	})

	err = discover.Register(context.Background(), "logv3", &discovery.Node{
		Host: "127.2.1.1",
		Port: 120104,
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	err = discover.Register(context.Background(), "logv3", &discovery.Node{
		Host: "127.2.1.1",
		Port: 120103,
	})
	if err != nil {
		t.Fatal(err)
		return
	}

	//err = discover.UnregisterAll(context.Background(), "logv4")
	//if err != nil {
	//	t.Fatal(err)
	//	return
	//}

	time.Sleep(time.Second)
	srv, err = discover.Discover(context.Background(), "logv3")
	if err != nil {
		t.Fatal(err)
		return
	}

	//discover.Unwatch()

	t.Logf("srvName:%s", srv.SrvName)
	for _, cfg := range srv.Nodes {
		t.Logf("node:%+v", cfg)

		err = discover.Unregister(context.Background(), "logv3", cfg, true)
		if err != nil {
			t.Fatal(err)
			return
		}
	}

	time.Sleep(time.Second)
	fmt.Println("discover all at:", time.Now().Unix())
	srv, err = discover.Discover(context.Background(), "logv3")
	if err != nil && err != discovery.ErrSrvNotFound {
		t.Fatal(err)
		return
	}

	t.Log(srv, err)

	t.Log("exit")
	time.Sleep(time.Minute)

	//srvs, err := discover.LoadAll(context.Background())
	//if err != nil {
	//	t.Fatal(err)
	//	return
	//}
	//
	//for _, srv := range srvs {
	//	t.Logf("srvName:%s", srv.SrvName)
	//	for _, cfg := range srv.Nodes {
	//		t.Logf("node:%+v", cfg)
	//		discover.Unregister(context.Background(), srv.SrvName, cfg, false)
	//	}
	//}
	//
	//srvs, err = discover.LoadAll(context.Background())
	//if err != nil {
	//	t.Fatal(err)
	//	return
	//}
	//
	//for _, srv := range srvs {
	//	t.Logf("srvName:%s", srv.SrvName)
	//	for _, cfg := range srv.Nodes {
	//		t.Logf("node:%+v", cfg)
	//		discover.Unregister(context.Background(), srv.SrvName, cfg, true)
	//	}
	//}
	//
	//srv, err = discover.Discover(context.Background(), srv.SrvName)
	//if err != nil {
	//	t.Fatal(err)
	//	return
	//}

	//for _, srv := range srvs {
	//	t.Logf("srvName:%s", srv.SrvName)
	//	for _, cfg := range srv.Nodes {
	//		t.Logf("node:%+v", cfg)
	//		discover.Unregister(context.Background(), srv.SrvName, cfg, false)
	//	}
	//}
	//srvs, err = discover.LoadAll(context.Background())
	//if err != nil {
	//	t.Fatal(err)
	//	return
	//}
	//for _, srv := range srvs {
	//	t.Logf("srvName:%s", srv.SrvName)
	//	for _, cfg := range srv.Nodes {
	//		t.Logf("node:%+v", cfg)
	//	}
	//}
}
