package nomats

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type upstream struct {
	target *url.URL

	// activeConns is used for least-conn tracking
	activeConns int64
	proxy       *httputil.ReverseProxy
}

type loadBalancer struct {
	hostname     string
	mu           sync.RWMutex
	upstreams    []*upstream
	currentIndex uint32 // For round-robin
	algorithm    string // Algorithm: "round-robin", "least-conn"

	httpSrv *http.Server
}

func newLoadBalancer(upstreams []string, algorithm string) *loadBalancer {
	var upstreamList []*upstream
	for _, target := range upstreams {
		u, err := url.Parse(target)
		if err != nil {
			log.Fatalf("Invalid upstream URL: %v", err)
		}
		upstreamList = append(upstreamList, &upstream{
			target:      u,
			activeConns: 0,
			proxy:       httputil.NewSingleHostReverseProxy(u),
		})
	}
	return &loadBalancer{
		upstreams: upstreamList,
		algorithm: algorithm,
	}
}

func (lb *loadBalancer) selectUpstream() *upstream {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.upstreams) == 0 {
		log.Fatalf("No upstreams available")
	}

	switch lb.algorithm {
	case "round-robin":
		return lb.selectRoundRobin()
	case "least-conn":
		return lb.selectLeastConn()
	default:
		log.Fatalf("Unknown algorithm: %s", lb.algorithm)
		return nil
	}
}

func (lb *loadBalancer) selectRoundRobin() *upstream {
	index := atomic.AddUint32(&lb.currentIndex, 1)
	return lb.upstreams[index%uint32(len(lb.upstreams))]
}

func (lb *loadBalancer) selectLeastConn() *upstream {
	var selected *upstream
	minConns := int64(^uint64(0) >> 1)
	for _, u := range lb.upstreams {
		if atomic.LoadInt64(&u.activeConns) <= minConns {
			selected = u
			minConns = atomic.LoadInt64(&u.activeConns)
		}
	}

	return selected
}

func (lb *loadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	upstream := lb.selectUpstream()

	// for least-conn accounting
	atomic.AddInt64(&upstream.activeConns, 1)
	defer atomic.AddInt64(&upstream.activeConns, -1)

	upstream.proxy.ServeHTTP(w, r)
}

// AddUpstream adds a new upstream to the given load balancer
// typically, the IP address of a Nomad allocation
func (lb *loadBalancer) AddUpstream(target string) error {
	u, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("Invalid upstream URL: %w", err)
	}

	newUpstream := &upstream{
		target:      u,
		activeConns: 0,
		proxy:       httputil.NewSingleHostReverseProxy(u),
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()
	// TODO: check for duplicates
	lb.upstreams = append(lb.upstreams, newUpstream)
	log.Printf("Added upstream: %s", target)

	return nil
}

// RemoveUpstream dynamically removes an upstream.
func (lb *loadBalancer) RemoveUpstream(target string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, u := range lb.upstreams {
		if u.target.String() == target {
			lb.upstreams = append(lb.upstreams[:i], lb.upstreams[i+1:]...)
			log.Printf("Removed upstream: %s", target)
			return
		}
	}
	log.Printf("Upstream not found: %s", target)
}

func (lb *loadBalancer) Start(ln net.Listener) error {

	lb.httpSrv = &http.Server{
		Handler: lb,
	}

	fmt.Println("http server started for", ln.Addr())
	if err := lb.httpSrv.Serve(ln); err != nil {
		return err
	}

	return nil
}

func (lb *loadBalancer) Stop(ctx context.Context) error {
	return lb.httpSrv.Shutdown(ctx)
}
