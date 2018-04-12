package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/medibloc/go-medibloc/medlet"
	log "github.com/medibloc/go-medibloc/util/logging"
	"github.com/urfave/cli"
)

var (
	version string
	commit  string
	branch  string
)

func main() {
	app := cli.NewApp()
	app.Action = medi
	app.Name = "medi"
	app.Usage = "medibloc command line interface"
	app.Version = versionStr()

	app.Run(os.Args)
}
func versionStr() string {
	if version == "" {
		return ""
	}
	return fmt.Sprintf("%s, branch %s, commit %s", version, branch, commit)
}

func medi(ctx *cli.Context) error {
	configFile := ctx.Args().Get(0)
	conf := medlet.LoadConfig(configFile)
	m, err := medlet.New(conf)
	if err != nil {
		return err
	}

	log.Init(m.Config().App.LogFile, m.Config().App.LogLevel, m.Config().App.LogAge)

	return runMedi(ctx, m)
}

func runMedi(ctx *cli.Context, m *medlet.Medlet) error {
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt, os.Kill)

	// Run Node
	log.Console().Info("Start medibloc...")

	m.Setup()
	m.Start()

	<-sigch
	m.Stop()
	log.Console().Info("Stop medibloc...")
	return nil
}
