package nomats

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"tailscale.com/tsnet"
)

// Server keeps a list of upstreams for a given nomad service
// and starts a tsnet server to proxy to upstreams
type Server struct {
	authKey string

	mu      sync.RWMutex
	proxies map[string]*loadBalancer
	servers map[string]*tsnet.Server

	dir        string
	listenAddr string
}

func New(cfg Config) *Server {
	return &Server{
		proxies:    map[string]*loadBalancer{},
		servers:    map[string]*tsnet.Server{},
		dir:        cfg.TailscaleDir,
		listenAddr: cfg.UpstreamListenAddr,
		authKey:    cfg.TailscaleAuthKey,
	}
}

func (s *Server) AddProxy(hostname string, remote string) error {
	s.mu.Lock()
	lb, found := s.proxies[hostname]
	s.mu.Unlock()

	if found {
		lb.AddUpstream(remote)
		return nil
	}

	log.Println(hostname, "was not found, adding it")

	newLB := newLoadBalancer([]string{}, "round-robin")
	newLB.AddUpstream(remote)

	os.MkdirAll(filepath.Join(s.dir, hostname), 0700)
	srv := &tsnet.Server{
		Hostname:  hostname,
		AuthKey:   s.authKey,
		Ephemeral: true,
		Dir:       filepath.Join(s.dir, hostname),
	}

	ln, err := srv.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}

	if _, found := s.proxies[hostname]; !found {
		s.mu.Lock()
		s.proxies[hostname] = newLB
		s.servers[hostname] = srv
		s.mu.Unlock()
	}

	// TODO: find a better way to handle this
	go func() {
		newLB.Start(ln)
	}()

	return nil
}

func (s *Server) DeleteProxy(hostname string) error {
	s.mu.Lock()
	lb, found := s.proxies[hostname]
	s.mu.Unlock()

	if found {
		delete(s.proxies, hostname)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()
		lb.Stop(ctx)
	}

	s.mu.Lock()
	srv, found := s.servers[hostname]
	s.mu.Unlock()
	if found {
		delete(s.servers, hostname)
		srv.Close()
	}

	return nil
}

func (s *Server) Start(ctx context.Context) error {
	for {
		<-ctx.Done()
		err := ctx.Err()
		log.Println(err)
		return err
	}
}

func (s *Server) Close() error {
	var multiErr error
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx := context.TODO()
	for name, p := range s.proxies {
		fmt.Println("deleting proxy:", name)
		if err := p.Stop(ctx); err != nil {
			errors.Join(multiErr, err)
		}
	}

	for name, s := range s.servers {
		fmt.Println("deleting tsnet:", name)
		if err := s.Close(); err != nil {
			errors.Join(multiErr, err)
		}
	}

	fmt.Println("deleting:", s.dir)
	if err := os.RemoveAll(s.dir); err != nil {
		errors.Join(multiErr, err)
		return err
	}

	return multiErr
}
