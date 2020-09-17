package leto

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/grandcat/zeroconf"
	"gopkg.in/yaml.v2"
)

type NodeLister struct {
	CacheDate time.Time       `yaml:"date"`
	Cache     map[string]Node `yaml:"nodes"`
}

type Node struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

func (n Node) RunMethod(name string, args, reply interface{}) error {
	c, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", n.Address, n.Port))
	if err != nil {
		return fmt.Errorf("Could not connect to '%s': %s", n.Name, err)
	}
	defer c.Close()
	return c.Call(name, args, reply)
}

func NewNodeLister() *NodeLister {
	res := &NodeLister{}
	res.load()
	return res
}

func (n *NodeLister) cacheFilePath() string {
	return filepath.Join(xdg.CacheHome, "fort/leto/node.cache")
}

func (n *NodeLister) load() {
	cachedData, err := ioutil.ReadFile(n.cacheFilePath())
	if err != nil {
		return
	}
	err = yaml.Unmarshal(cachedData, n)
	if err != nil {
		n.CacheDate = time.Now().Add(-10 * time.Hour)
	}
}

func (n *NodeLister) save() {
	if err := os.MkdirAll(filepath.Dir(n.cacheFilePath()), 0755); err != nil {
		return
	}
	yamlData, err := yaml.Marshal(n)
	if err != nil {
		return
	}

	ioutil.WriteFile(n.cacheFilePath(), yamlData, 0644)
}

func (n *NodeLister) ListNodes() (map[string]Node, error) {
	if time.Now().Before(n.CacheDate.Add(NODE_CACHE_TTL)) == true {
		return n.Cache, nil
	}

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("Could not create resolver: %s", err)
	}
	entries := make(chan *zeroconf.ServiceEntry, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err = resolver.Browse(ctx, "_leto._tcp", "local.", entries)
	if err != nil {
		return nil, fmt.Errorf("Could not browse for leto instances: %s", err)
	}

	<-ctx.Done()

	res := make(map[string]Node)

	for e := range entries {
		name := strings.TrimPrefix(e.Instance, "leto.")
		address := strings.TrimSuffix(e.HostName, ".")
		port := e.Port
		res[name] = Node{Name: name, Address: address, Port: port}
	}
	n.Cache = res
	n.CacheDate = time.Now()

	n.save()

	return res, nil

}
