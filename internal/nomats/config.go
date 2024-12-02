package nomats

type Config struct {
	NomadAddr          string `env:"NOMAD_ADDR"`
	UpstreamListenAddr string `env:"UPSTREAM_ADDR" envDefault:":8080"`
	TailscaleDir       string `env:"TAILSCALE_DIR" envDefault:"/tmp/tailscale/data"`
}
