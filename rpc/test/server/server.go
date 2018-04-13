package main

import (
	"os"
	"os/signal"

	"github.com/medibloc/go-medibloc/rpc"
)

func main() {
	sigch := make(chan os.Signal)
	signal.Notify(sigch, os.Interrupt, os.Kill)
	server := rpc.NewServer()
	server.Start("localhost:10000")
	defer server.Stop()
	server.RunGateway()
	<-sigch
}
