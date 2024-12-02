# nomats

nomats, portmanteau of (Hashicorp) Nomad and Tailscale (ts), is a tsnet http reverse proxy for nomad services.

For every service in nomad a new tsnet is spun up which proxies http requests to jobs running in Nomad.
