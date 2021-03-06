# Host Metrics Receiver

The Host Metrics receiver generates metrics about the host system scraped
from various sources. This is intended to be used when the collector is
deployed as an agent.

If you are only interested in a subset of metrics from a particular source,
it is recommended you use this receiver with the
[Filter Processor](https://github.com/open-telemetry/opentelemetry-collector/tree/master/processor/filterprocessor).

## Configuration

The collection interval and the categories of metrics to be scraped can be
configured:

```yaml
hostmetrics:
  collection_interval: <duration> # default = 1m
  scrapers:
    <scraper1>:
    <scraper2>:
    ...
```

If you would like to scrape some metrics at a different frequency than others,
you can configure multiple `hostmetrics` receivers with different
`collection_interval` values. For example:

```yaml
receivers:
  hostmetrics:
    collection_interval: 30s
    scrapers:
      cpu:
      memory:

  hostmetrics/disk:
    collection_interval: 1m
    scrapers:
      disk:
      filesystem:

service:
  pipelines:
    metrics:
      receivers: [hostmetrics, hostmetrics/disk]
```

## Scrapers

The available scrapers are:

Scraper    | Supported OSs      | Description 
-----------|--------------------|-------------
cpu        | All                | CPU utilization metrics
disk       | All                | Disk I/O metrics
load       | All                | CPU load metrics
filesystem | All                | File System utilization metrics
memory     | All                | Memory utilization metrics
network    | All                | Network interface I/O metrics & TCP connection metrics
processes  | Linux              | Process count metrics
swap       | All                | Swap space utilization and I/O metrics
process    | Linux & Windows    | Per process CPU, Memory, and Disk I/O metrics

Several scrapers support additional configuration:

#### Disk

```yaml
disk:
  <include|exclude>:
    devices: [ <device name>, ... ]
    match_type: <strict|regexp>
```

#### File System

```yaml
filesystem:
  <include|exclude>:
    devices: [ <device name>, ... ]
    match_type: <strict|regexp>
```

#### Network

```yaml
network:
  <include|exclude>:
    interfaces: [ <interface name>, ... ]
    match_type: <strict|regexp>
```

#### Process

```yaml
process:
  disk:
    <include|exclude>:
      names: [ <process name>, ... ]
      match_type: <strict|regexp>
```
