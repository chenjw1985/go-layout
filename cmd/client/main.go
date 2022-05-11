package main

import (
	"context"
	"fmt"

	"github.com/davidchen-cn/go-layout/api/helloworld/v1"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	// new etcd client
	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	})
	if err != nil {
		panic(err)
	}
	// new dis with etcd client
	dis := etcd.New(client)

	endpoint := "discovery:///go-layout"
	conn, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint(endpoint), grpc.WithDiscovery(dis))
	if err != nil {
		panic(err)
	}
	c := v1.NewGreeterClient(conn)
	reply, err := c.SayHello(context.Background(), &v1.HelloRequest{Name: "GRPC Client"})
	if err != nil {
		panic(err)
	}
	fmt.Println(reply)
}
