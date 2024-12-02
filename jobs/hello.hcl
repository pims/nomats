job "hello" {
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
    count = 2

    network {
      port "www" {} // dynamic port
    }

    service {
      provider = "nomad"
      port     = "www"
      tags = ["tailscale.com/enabled=true"]
    }

    
    task "web" {
      driver = "raw_exec"

      config {

        command = "/usr/bin/statik"
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