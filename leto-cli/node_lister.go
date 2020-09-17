package main

import (
	"context"
	"fmt"
	"net/rpc"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

type NodeLister struct {
	cacheDate time.Time
	cache     map[string]Node
}

type Node struct {
	Name    string
	Address string
}

func (n Node) RunMethod(name string, args, reply interface{}) error {
	c, err := rpc.DialHTTP("tcp", n.Address)
	if err != nil {
		return fmt.Errorf("Could not connect to '%s': %s", n.Name, err)
	}
	defer c.Close()
	return c.Call(name, args, reply)
}

func NewNodeLister() *NodeLister {
	return &NodeLister{}
}

func (n *NodeLister) ListNodes() (map[string]Node, error) {
	if time.Now().Before(n.cacheDate.Add(2*time.Minute)) == true {
		return n.cache, nil
	}

	resolver, err := zeroconf.NewResolver(nil)
	entries := make(chan *zeroconf.ServiceEntry, 100)
	errors := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		err = resolver.Browse(ctx, "_leto._tcp", "local.", entries)
		if err != nil {
			errors <- err
		}
		close(errors)
	}()
	res := make(map[string]Node)
	err, ok := <-errors
	if ok == true {
		return nil, fmt.Errorf("Could not browse for leto instances: %s", err)
	}
	<-errors

	for e := range entries {
		name := strings.TrimPrefix(e.Instance, "leto.")
		address := fmt.Sprintf("%s:%d", strings.TrimSuffix(e.HostName, "."), e.Port)
		res[name] = Node{Name: name, Address: address}
	}

	return res, nil

}
