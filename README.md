# OPI storage gRPC to nvidia SPDK json-rpc bridge

[![Linters](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/linters.yml/badge.svg)](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/linters.yml)
[![tests](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/go.yml/badge.svg)](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/go.yml)
[![Docker](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/docker-publish.yml)
[![License](https://img.shields.io/github/license/opiproject/opi-nvidia-bridge?style=flat-square&color=blue&label=License)](https://github.com/opiproject/opi-nvidia-bridge/blob/master/LICENSE)
[![codecov](https://codecov.io/gh/opiproject/opi-nvidia-bridge/branch/main/graph/badge.svg)](https://codecov.io/gh/opiproject/opi-nvidia-bridge)
[![Go Report Card](https://goreportcard.com/badge/github.com/opiproject/opi-nvidia-bridge)](https://goreportcard.com/report/github.com/opiproject/opi-nvidia-bridge)
[![Last Release](https://img.shields.io/github/v/release/opiproject/opi-nvidia-bridge?label=Latest&style=flat-square&logo=go)](https://github.com/opiproject/opi-nvidia-bridge/releases)

This is a nvidia plugin to OPI storage APIs based on SPDK.

## I Want To Contribute

This project welcomes contributions and suggestions.  We are happy to have the Community involved via submission of **Issues and Pull Requests** (with substantive content or even just fixes). We are hoping for the documents, test framework, etc. to become a community process with active engagement.  PRs can be reviewed by by any number of people, and a maintainer may accept.

See [CONTRIBUTING](https://github.com/opiproject/opi/blob/main/CONTRIBUTING.md) and [GitHub Basic Process](https://github.com/opiproject/opi/blob/main/doc-github-rules.md) for more details.

## Getting started

```bash
go build -v -buildmode=plugin -o /opi-nvidia-bridge.so ./...
```

 in main app:

```go
package main
import (
    "plugin"
    pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
)
func main()
    plug, err := plugin.Open("/opi-nvidia-bridge.so")
    feNvmeSymbol, err := plug.Lookup("PluginFrontendNvme")
    var feNvme pb.FrontendNvmeServiceServer
    feNvme, ok := feNvmeSymbol.(pb.FrontendNvmeServiceServer)
    s := grpc.NewServer()
    pb.RegisterFrontendNvmeServiceServer(s, feNvme)
    reflection.Register(s)
}
```
