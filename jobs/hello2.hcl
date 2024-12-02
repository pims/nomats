job "hello2" {
  // Specifies the datacenter where this job should be run
  // This can be omitted and it will default to ["*"]
  datacenters = ["*"]

  meta {
    // User-defined key/value pairs that can be used in your jobs.
    // You can also use this meta block within Group and Task levels.
    foo = "baz"
  }

  // A group defines a series of tasks that should be co-located
  // on the same client (host). All tasks within a group will be
  // placed on the same host.
  group "servers" {

    // Specifies the number of instances of this group that should be running.
    // Use this to scale or parallelize your job.
    // This can be omitted and it will default to 1.
    count = 1

    network {
      port "www" {}
    }

    service {
      provider = "nomad"
      port     = "www"
      tags = ["tailscale.com/enabled=false"]
    }

    // Tasks are individual units of work that are run by Nomad.
    task "web" {
      // This particular task starts a simple web server within a Docker container
      driver = "raw_exec"

      config {

        command = "/usr/local/bin/statik"
        args    = [":${NOMAD_PORT_www}"]
      }

      // Specify the maximum resources required to run the task
      resources {
        cpu    = 50
        memory = 64
      }
    }
  }
}