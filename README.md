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
func main() {
    plug, err := plugin.Open("/opi-nvidia-bridge.so")
    feNvmeSymbol, err := plug.Lookup("PluginFrontendNvme")
    var feNvme pb.FrontendNvmeServiceServer
    feNvme, ok := feNvmeSymbol.(pb.FrontendNvmeServiceServer)
    s := grpc.NewServer()
    pb.RegisterFrontendNvmeServiceServer(s, feNvme)
    reflection.Register(s)
}
```

## Using docker

on DPU/IPU (i.e. with IP=10.10.10.1) run

```bash
$ docker run --rm -it -v /var/tmp/:/var/tmp/ -p 50051:50051 ghcr.io/opiproject/opi-nvidia-bridge:main
2022/11/29 00:03:55 plugin serevr is &{{}}
2022/11/29 00:03:55 server listening at [::]:50051
```

on X86 management VM run

```bash
docker run --network=host --rm -it namely/grpc-cli ls   --json_input --json_output 10.10.10.10:50051 -l
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNVMeSubsystem "{subsystem : {spec : {id : {value : 'subsystem2'}, nqn: 'nqn.2022-09.io.spdk:opitest2', serial_number: 'myserial2', model_number: 'mymodel2', max_namespaces: 11} } }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNVMeSubsystem "{}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNVMeSubsystem "{subsystem_id : {value : 'subsystem2'} }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNVMeController "{controller : {spec : {id : {value : 'controller1'}, nvme_controller_id: 2, subsystem_id : { value : 'subsystem2' }, pcie_id : {physical_function : 0}, max_nsq:5, max_ncq:5 } } }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNVMeController "{subsystem_id : { value : 'subsystem2' }}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNVMeController "{controller_id : {value : 'controller1'} }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNVMeNamespace "{namespace : {spec : {id : {value : 'namespace1'}, subsystem_id : { value : 'subsystem2' }, controller_id : { value : 'controller1' }, volume_id : { value : 'Malloc0' }, 'host_nsid' : '1', uuid:{value : '1b4e28ba-2fa1-11d2-883f-b9a761bde3fb'}, nguid: '1b4e28ba-2fa1-11d2-883f-b9a761bde3fb', eui64: 1967554867335598546 } } }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNVMeNamespace "{subsystem_id : { value : 'subsystem2' } }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNVMeNamespace "{namespace_id : {value : 'namespace1'} }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.1:50051 NVMfRemoteControllerConnect "{ctrl : {id: '12', traddr:'11.11.11.2', subnqn:'nqn.2016-06.com.opi.spdk.target0', trsvcid:'4444', trtype:'NVME_TRANSPORT_TCP', adrfam:'NVMF_ADRFAM_IPV4', hostnqn:'nqn.2014-08.org.nvmexpress:uuid:feb98abe-d51f-40c8-b348-2753f3571d3c'}}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.1:50051 NVMfRemoteControllerGet "{'id': '12'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.1:50051 NVMfRemoteControllerDisconnect "{'id': '12'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNVMeNamespace "{namespace_id : {value : 'namespace1'} }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNVMeController "{controller_id : {value : 'controller1'} }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNVMeSubsystem "{subsystem_id : {value : 'subsystem2'} }"
```
