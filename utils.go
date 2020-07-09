package leto

import (
	"context"
	"fmt"
	"net/rpc"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

func RunForHost(host, name string, args interface{}, reply interface{}) (string, int, error) {

	resolver, err := zeroconf.NewResolver(nil)
	entries := make(chan *zeroconf.ServiceEntry)
	errors := make(chan error)
	found := false
	hostname := ""
	port := 0
	go func(results <-chan *zeroconf.ServiceEntry) {
		defer func() { close(errors) }()
		for e := range results {
			found = true
			hostname = strings.TrimSuffix(e.HostName, ".")
			port = e.Port
			client, err := rpc.DialHTTP("tcp",
				fmt.Sprintf("%s:%d", hostname, port))
			if err != nil {
				errors <- err
				return
			}
			errors <- client.Call(name, args, reply)
			return
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = resolver.Lookup(ctx, host, "_leto._tcp", "local.", entries)
	if err != nil {
		return hostname, port, err
	}
	err, ok := <-errors
	if ok == true {
		return hostname, port, err
	}
	if found == false {
		return hostname, port, fmt.Errorf("Could not found host '%s'", host)
	}
	return hostname, port, nil
}
