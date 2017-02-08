package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/XiaoweiQian/macvlan-driver/drivers"
	"github.com/codegangsta/cli"
	pluginNet "github.com/docker/go-plugins-helpers/network"
)

const (
	version     = "0.1"
	networkType = "macvlan_swarm"
)

func main() {

	var flagDebug = cli.BoolFlag{
		Name:  "debug, d",
		Usage: "enable debugging",
	}
	app := cli.NewApp()
	app.Name = "docker-macvlan"
	app.Usage = "Docker Macvlan Networking"
	app.Version = version
	app.Flags = []cli.Flag{
		flagDebug,
	}
	app.Action = Run
	app.Run(os.Args)
}

// Run initializes the driver
func Run(ctx *cli.Context) {
	if ctx.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	d, err := drivers.Init(nil)
	if err != nil {
		panic(err)
	}
	h := pluginNet.NewHandler(d)
	h.ServeUnix("root", networkType)
}
