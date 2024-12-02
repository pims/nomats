package nomad

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"github.com/hashicorp/nomad/api"
	"github.com/pims/nomats/internal/nomats"
)

const (
	tailscaleTag = "tailscale.com/enabled=true"
)

func Watch(ctx context.Context, client *api.Client, srv *nomats.Server) error {
	topics := map[api.Topic][]string{
		api.TopicService: {"*"},
	}

	st, meta, err := client.Services().List(nil)
	if err != nil {
		return err
	}
	lastIdx := meta.LastIndex

	for _, regList := range st {
		for _, stub := range regList.Services {
			if slices.Contains(stub.Tags, tailscaleTag) {
				registrations, _, err := client.Services().Get(stub.ServiceName, nil)
				if err != nil {
					return err
				}
				fmt.Println(len(registrations), "registrations")
				for _, registration := range registrations {

					target := fmt.Sprintf("http://%s:%d", registration.Address, registration.Port)
					srv.AddProxy(registration.ServiceName, target)
					fmt.Println("done adding", target)
				}
			}
		}
	}

	// now watch for changes that might have happened since we initially fetched the list of services
	svcChan, err := client.EventStream().Stream(ctx, topics, lastIdx+1, nil)
	if err != nil {
		log.Println(err)
		return err
	}

	for {
		select {
		case devent := <-svcChan:
			for _, evt := range devent.Events {
				b, _ := json.Marshal(evt)
				fmt.Println(string(b))
				s, err := evt.Service()
				if err != nil {
					return err
				}

				target := fmt.Sprintf("http://%s:%d", s.Address, s.Port)
				switch evt.Type {
				case "ServiceDeregistration":
					log.Println("deregistering", s.ServiceName, target)
					srv.DeleteProxy(s.ServiceName)
				case "ServiceRegistration":
					// skip services that don't have the tailscale tag
					if slices.Contains(s.Tags, tailscaleTag) {
						log.Println("registering", s.ServiceName, target)
						srv.AddProxy(s.ServiceName, target)
					}
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

}
