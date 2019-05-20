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
)

type Leto struct {
	artemis *ArtemisManager
}

func (l *Leto) StartTracking(args *leto.TrackingStart, resp *leto.Response) error {
	err := l.artemis.Start(args)
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Error = ""
	}
	return nil
}

func (l *Leto) StopTracking(args *leto.TrackingStop, resp *leto.Response) error {
	err := l.artemis.Stop()
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Error = ""
	}
	return nil
}

func Execute() error {
	l := &Leto{}
	var err error
	l.artemis, err = NewArtemisManager()
	if err != nil {
		return err
	}
	logger := log.New(os.Stderr, "[rpc]", log.LstdFlags)
	rpcRouter := rpc.NewServer()
	rpcRouter.Register(l)
	rpcRouter.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	rpcServer := http.Server{
		Addr:    fmt.Sprintf(":%d", LETO_PORT),
		Handler: rpcRouter,
	}

	idleConnections := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := rpcServer.Shutdown(context.Background()); err != nil {
			logger.Printf("could not shutdown: %s", err)
		}
		close(idleConnections)
	}()
	logger.Printf("listening on %s", rpcServer.Addr)
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
