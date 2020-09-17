package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"

	"github.com/formicidae-tracker/leto"
	"github.com/grandcat/zeroconf"
)

type Leto struct {
	artemis *ArtemisManager
	logger  *log.Logger
}

func (l *Leto) StartTracking(args *leto.TrackingConfiguration, resp *leto.Response) error {
	l.logger.Printf("new start request for experiment '%s'", args.ExperimentName)
	err := l.artemis.Start(args)
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Error = ""
	}
	return nil
}

func (l *Leto) StopTracking(args *leto.NoArgs, resp *leto.Response) error {
	l.logger.Printf("new stop request")
	err := l.artemis.Stop()
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Error = ""
	}
	return nil
}

func (l *Leto) Status(args *leto.NoArgs, resp *leto.Status) error {
	*resp = l.artemis.Status()
	return nil
}

func (l *Leto) Link(args *leto.Link, resp *leto.Response) error {
	var err error = nil
	defer func() {
		resp.Error = ""
		if err != nil {
			resp.Error = err.Error()
		}
	}()
	host, err := os.Hostname()
	if err != nil {
		return nil
	}

	host = "leto." + host

	if args.Master != host && args.Slave != host {
		err = fmt.Errorf("Host %s is neither master (%s) or slave (%s)", host, args.Master, args.Slave)
		return nil
	}

	if args.Slave == host {
		err = l.artemis.SetMaster(args.Master)
		return nil
	}
	respSlave := leto.Response{}
	_, _, err = leto.RunForHost(args.Slave, "Leto.Link", args, &respSlave)
	if err != nil {
		return nil
	}
	err = respSlave.ToError()
	if err != nil {
		return nil
	}

	err = l.artemis.AddSlave(args.Slave)
	return nil
}

func (l *Leto) Unlink(args *leto.Link, resp *leto.Response) error {
	var err error = nil
	defer func() {
		resp.Error = ""
		if err != nil {
			resp.Error = err.Error()
		}
	}()
	host, err := os.Hostname()
	if err != nil {
		return nil
	}

	host = "leto." + host

	if args.Master != host && args.Slave != host {
		err = fmt.Errorf("Host %s is neither master (%s) or slave (%s)", host, args.Master, args.Slave)
		return nil
	}

	if args.Slave == host {
		err = l.artemis.SetMaster("")
		return nil
	}

	err = l.artemis.RemoveSlave(args.Slave)
	if err != nil {
		return nil
	}

	respSlave := leto.Response{}
	_, _, err = leto.RunForHost(args.Slave, "Leto.Unlink", args, &respSlave)
	if err != nil {
		return nil
	}
	err = respSlave.ToError()
	return nil
}

func Execute() error {
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	l := &Leto{}
	l.artemis, err = NewArtemisManager()
	if err != nil {
		return err
	}
	l.logger = log.New(os.Stderr, "[rpc] ", log.LstdFlags)
	rpcRouter := rpc.NewServer()
	rpcRouter.Register(l)
	rpcRouter.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	rpcServer := http.Server{
		Addr:    fmt.Sprintf(":%d", leto.LETO_PORT),
		Handler: rpcRouter,
	}

	idleConnections := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := rpcServer.Shutdown(context.Background()); err != nil {
			l.logger.Printf("could not shutdown: %s", err)
		}
		close(idleConnections)
	}()

	go func() {
		server, err := zeroconf.Register("leto."+host, "_leto._tcp", "local.", leto.LETO_PORT, nil, nil)
		if err != nil {
			log.Printf("[avahi] register error: %s", err)
			return
		}
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		server.Shutdown()
	}()

	l.logger.Printf("listening on %s", rpcServer.Addr)
	if err := rpcServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	<-idleConnections

	return nil
}

func main() {
	if err := Execute(); err != nil {
		log.Fatalf("Unhandled error: %s", err)
	}
}
