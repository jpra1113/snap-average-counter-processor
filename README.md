# snap-average-counter-processor

Plugin for snap

## Getting started

This project requires Go to be installed. On OS X with Homebrew you can just run `brew install go`.

Running it then should be as simple as:

```console
$ make
```

### Testing
#### Unit test:
``make test``

#### Integrating test:
this will launch a snap stack with one collector, average-counter-processor and influxdb publisher, you can update ``test-task.yaml`` to see this plugin work properly.
``make run-si-test``

clean integrating test by ``make cleanup-si-test``

### Configuration:
``collect.namespaces``: Collect metrics by namespaces, separate by comma, e.g. "default, hyperpilot"

``collect.include_empty_namespace``: Bool value, set to True if there is no "namespace" field or "namespace" has empty value

``collect.exclude_metrics``: metrics which you don't want to collect, it can use wildcard(*) value and separate by comma, e.g. "intel/docker/*/mem, intel/procfs/perc*"

``collect.exclude_metrics.except``: except list for exclude_metrics and separate by comma, e.g. "*precentage, intel/docker/test/mem"

``average``: list of metrics which should calculate the rate (diff per sec), and wildcard(*) is support, e.g. "*/io_time, */read_time, */reads_completed"

### Example
```
---
  version: 1
  schedule:
    type: "simple"
    interval: "1s"
  workflow:
    collect:
      metrics:
        # /intel/docker/*: {}
        /intel/procfs/meminfo/*: {}
        /intel/procfs/disk/*: {}
        /intel/procfs/cpu/*: {}
      config:
        /intel/procfs:
          proc_path": "/proc"
        /intel/docker:
          endpoint: "unix:///var/run/docker.sock"
      process:
        - plugin_name: "snap-average-counter-processor"
          config:
            collect.namespaces: "default, hyperpilot"
            collect.include_empty_namespace: true
            collect.exclude_metrics: "intel/procfs/*, intel/docker/stats/*"
            collect.exclude_metrics.except: "*perc, *percentage"
            average: "*"
          publish:
            - plugin_name: "influxdb"
              config:
                host: "influxsrv"
                port: 8086
                database: "snap"
                user: "root"
                password: "default"
                https: false
                skip-verify: false

```
