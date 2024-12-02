# nomats

nomats, portmanteau of (Hashicorp) Nomad and Tailscale (ts), is a tsnet http reverse proxy for nomad services.

For every service in nomad a new tsnet is spun up which proxies http requests to jobs running in Nomad.

> [!WARNING]
> This is highly experimental, known to have bugs and should not be used in production
> It was fun to tinker with Tailscale though :)

## Background

I like to run simple Go binaries with Nomad, most of the time on a single host.
I like using Tailscale.
I wish I could quickly connect to services running in Nomad from device connected via Tailscale.
Now I can :)

## Getting started

### Start Nomad

```
nomad agent -dev -config nomad.conf
```

```nomad.conf
data_dir  = "/tmp/nomad" # changeme


plugin "raw_exec" {
  config {
    enabled = true
  }
}

server {
  enabled          = true
  bootstrap_expect = 1
}

client {
  enabled       = true
}

```

### Run nomats

Run `nomats`, on the same node, or specify the nomad address via the NOMAD_ADDR environment variable (auth not supported yet)

```
nomats
```

### Run a Nomad job

```
nomad job run jobs/hello.hcl
```

the key part being:

```hcl
service {
    provider = "nomad"
    port     = "www"
    tags = ["tailscale.com/enabled=true"] 👈 the magic ✨
}
```

```
tim@laptop nomats % nomad job run jobs/hello.hcl
==> 2024-12-01T20:14:53-08:00: Monitoring evaluation "d9822d1f"
2024-12-01T20:14:53-08:00: Evaluation triggered by job "hello"
2024-12-01T20:14:53-08:00: Allocation "3106f10a" created: node "441d5659", group "servers"
2024-12-01T20:14:54-08:00: Evaluation within deployment: "3ab7dde2"
2024-12-01T20:14:54-08:00: Allocation "3106f10a" status changed: "pending" -> "running" (Tasks are running)
2024-12-01T20:14:54-08:00: Evaluation status changed: "pending" -> "complete"
==> 2024-12-01T20:14:54-08:00: Evaluation "d9822d1f" finished with status "complete"
==> 2024-12-01T20:14:54-08:00: Monitoring deployment "3ab7dde2"
✓ Deployment "3ab7dde2" successful
```

### Connect to that service via Tailscale's Magic DNS

```
curl -4 http://hello-servers:8080

{"NOMAD_ADDR_www":"127.0.0.1:24028","NOMAD_ALLOC_DIR":"/tmp/nomad/alloc/d96af0be-747b-91ac-325e-9565b4797e8e/alloc","NOMAD_ALLOC_ID":"d96af0be-747b-91ac-325e-9565b4797e8e","NOMAD_ALLOC_INDEX":"0","NOMAD_ALLOC_NAME":"hello.servers[0]","NOMAD_ALLOC_PORT_www":"24028","NOMAD_CPU_LIMIT":"50","NOMAD_DC":"dc1","NOMAD_GROUP_NAME":"servers","NOMAD_HOST_ADDR_www":"127.0.0.1:24028","NOMAD_HOST_IP_www":"127.0.0.1","NOMAD_HOST_PORT_www":"24028","NOMAD_IP_www":"127.0.0.1","NOMAD_JOB_ID":"hello","NOMAD_JOB_NAME":"hello","NOMAD_MEMORY_LIMIT":"64","NOMAD_META_FOO":"baz","NOMAD_META_foo":"baz","NOMAD_NAMESPACE":"default","NOMAD_PORT_www":"24028","NOMAD_REGION":"global","NOMAD_SECRETS_DIR":"/tmp/nomad/alloc/d96af0be-747b-91ac-325e-9565b4797e8e/web/secrets","NOMAD_SHORT_ALLOC_ID":"d96af0be","NOMAD_TASK_DIR":"/tmp/nomad/alloc/d96af0be-747b-91ac-325e-9565b4797e8e/web/local","NOMAD_TASK_NAME":"web"}
```
