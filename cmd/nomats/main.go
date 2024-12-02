package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v11"
	"github.com/hashicorp/nomad/api"
	"github.com/pims/nomats/internal/nomad"
	"github.com/pims/nomats/internal/nomats"
	"golang.org/x/sync/errgroup"
)

func main() {

	var cfg nomats.Config
	if err := env.Parse(&cfg); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer done()
	g, gctx := errgroup.WithContext(ctx)

	srv := nomats.New(cfg.TailscaleDir, cfg.UpstreamListenAddr)

	nomadCfg := api.DefaultConfig()
	nomadCfg.Address = cfg.NomadAddr
	api, err := api.NewClient(nomadCfg)
	if err != nil {
		log.Fatal(err)
	}

	g.Go(func() error {
		log.Println("watching for events")
		if err := nomad.Watch(gctx, api, srv); err != nil {
			log.Println(err)
			return err
		}
		return nil
	})

	g.Go(func() error {
		return srv.Start(gctx)
	})

	if err := g.Wait(); err != nil {
		srv.Close()
	}

}
