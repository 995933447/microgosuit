package test

import (
	"context"
	"fmt"
	"github.com/etcd-io/etcd/clientv3"
	"github.com/etcd-io/etcd/mvcc/mvccpb"
	"strconv"
	"testing"
	"time"
)

const watchPrefix = "/a/b"

func TestEtcdWatch(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:12379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	go func() {
		wCh := cli.Watch(context.Background(), watchPrefix, clientv3.WithPrefix())
		for {
			data := <-wCh
			fmt.Println("etcd watch data", data)
			for i, event := range data.Events { // 每次有几个事件，这里就循环几次
				fmt.Printf("receive %v event, %v\n", i, event.Type) // event.Type表示事件类型，PUT还是DELETE
				switch event.Type {
				case clientv3.EventTypePut:
					// do your sth
				case clientv3.EventTypeDelete:
					// do your sth
				}

				getKey, getValue := getKeyAndVal(event)
				fmt.Printf("已监听到信号, %v=%v 进行了 %v 操作. 下一步通知操作方做相应处理\n", getKey, getValue, event.Type)

				// 实际业务处理...
			}
		}
	}()

	for i := 0; ; i++ {
		fmt.Println("i=", i)
		if i%3 == 0 { // 模拟触发某个条件，则开始触发watch
			key := fmt.Sprintf("%s/%v", watchPrefix, i)
			if i%2 == 0 { // 假设是偶数的话进行DELETE操作（纯delete操作时注释掉下面的else）
				fmt.Printf("触发了条件，准备DELETE: %v\n", key)
				_, err = cli.Delete(context.Background(), key)
				if err != nil {
					fmt.Printf("DELETE 失败： %v\n", err)
					continue
				}

				fmt.Printf("DELETE %v success\n", key)
			} else {
				fmt.Printf("触发了条件，准备PUT: %v\n", key)
				_, err = cli.Put(context.Background(), key, strconv.Itoa(i))
				if err != nil {
					fmt.Printf("PUT 失败： %v\n", err)
					continue
				}

				fmt.Printf("PUT %v success\n", key)
			}
		}

		time.Sleep(time.Second * 1)
	}
}

func getKeyAndVal(event *clientv3.Event) (getKey, getValue string) {
	fmt.Printf("event.PrevKv==nil结果 %v, event.Kv==nil结果 %v\n", event.PrevKv == nil, event.Kv == nil)
	fmt.Printf("event.PrevKv= %+v\n", event.PrevKv)
	fmt.Printf("event.Kv= %+v\n", event.Kv)
	if event.PrevKv != nil {
		getKey, getValue = string(event.PrevKv.Key), string(event.PrevKv.Value)
	}

	if event.Type == mvccpb.PUT {
		if event.Kv != nil {
			getKey, getValue = string(event.Kv.Key), string(event.Kv.Value)
		}
	}

	return
}
