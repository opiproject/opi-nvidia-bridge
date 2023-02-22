# OPI storage gRPC to nvidia SPDK json-rpc bridge

[![Linters](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/linters.yml/badge.svg)](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/linters.yml)
[![tests](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/go.yml/badge.svg)](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/go.yml)
[![Docker](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/docker-publish.yml)
[![License](https://img.shields.io/github/license/opiproject/opi-nvidia-bridge?style=flat-square&color=blue&label=License)](https://github.com/opiproject/opi-nvidia-bridge/blob/master/LICENSE)
[![codecov](https://codecov.io/gh/opiproject/opi-nvidia-bridge/branch/main/graph/badge.svg)](https://codecov.io/gh/opiproject/opi-nvidia-bridge)
[![Go Report Card](https://goreportcard.com/badge/github.com/opiproject/opi-nvidia-bridge)](https://goreportcard.com/report/github.com/opiproject/opi-nvidia-bridge)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/opiproject/opi-nvidia-bridge)
[![Pulls](https://img.shields.io/docker/pulls/opiproject/opi-nvidia-bridge.svg?logo=docker&style=flat&label=Pulls)](https://hub.docker.com/r/opiproject/opi-nvidia-bridge)
[![Last Release](https://img.shields.io/github/v/release/opiproject/opi-nvidia-bridge?label=Latest&style=flat-square&logo=go)](https://github.com/opiproject/opi-nvidia-bridge/releases)

This is a nvidia plugin to OPI storage APIs based on SPDK.

## I Want To Contribute

This project welcomes contributions and suggestions.  We are happy to have the Community involved via submission of **Issues and Pull Requests** (with substantive content or even just fixes). We are hoping for the documents, test framework, etc. to become a community process with active engagement.  PRs can be reviewed by by any number of people, and a maintainer may accept.

See [CONTRIBUTING](https://github.com/opiproject/opi/blob/main/CONTRIBUTING.md) and [GitHub Basic Process](https://github.com/opiproject/opi/blob/main/doc-github-rules.md) for more details.

## Getting started

build like this:

```bash
go build -v -o /opi-nvidia-bridge ./cmd/...
```

import like this:

```go
import "github.com/opiproject/opi-nvidia-bridge/pkg/frontend"
```

## Using docker

on DPU/IPU (i.e. with IP=10.10.10.1) run

```bash
$ docker run --rm -it -v /var/tmp/:/var/tmp/ -p 50051:50051 ghcr.io/opiproject/opi-nvidia-bridge:main
2022/11/29 00:03:55 plugin serevr is &{{}}
2022/11/29 00:03:55 server listening at [::]:50051
```

on X86 management VM run

reflection

```bash
$ docker run --network=host --rm -it namely/grpc-cli ls --json_input --json_output 10.10.10.10:50051 -l
grpc.reflection.v1alpha.ServerReflection
opi_api.storage.v1.AioControllerService
opi_api.storage.v1.FrontendNvmeService
opi_api.storage.v1.FrontendVirtioBlkService
opi_api.storage.v1.FrontendVirtioScsiService
opi_api.inventory.v1.InventorySvc;
opi_api.storage.v1.MiddleendService
opi_api.storage.v1.NVMfRemoteControllerService
opi_api.storage.v1.NullDebugService
```

full test suite

```bash
docker run --rm -it --network=host docker.io/opiproject/godpu:main storagetest --addr="10.10.10.10:50051"
```

or manually

```bash
docker run --network=host --rm -it namely/grpc-cli ls   --json_input --json_output 10.10.10.10:50051 -l
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNVMeSubsystem "{nv_me_subsystem : {spec : {id : {value : 'subsystem2'}, nqn: 'nqn.2022-09.io.spdk:opitest2', serial_number: 'myserial2', model_number: 'mymodel2', max_namespaces: 11} } }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNVMeSubsystems "{}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNVMeSubsystem "{name : 'subsystem2'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNVMeController "{nv_me_controller : {spec : {id : {value : 'controller1'}, nvme_controller_id: 2, subsystem_id : { value : 'subsystem2' }, pcie_id : {physical_function : 0}, max_nsq:5, max_ncq:5 } } }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNVMeControllers "{parent : 'subsystem2'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNVMeController "{name : 'controller1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNVMeNamespace "{nv_me_namespace : {spec : {id : {value : 'namespace1'}, subsystem_id : { value : 'subsystem2' }, volume_id : { value : 'Malloc0' }, 'host_nsid' : '10', uuid:{value : '1b4e28ba-2fa1-11d2-883f-b9a761bde3fb'}, nguid: '1b4e28ba-2fa1-11d2-883f-b9a761bde3fb', eui64: 1967554867335598546 } } }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNVMeNamespaces "{parent : 'subsystem2'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNVMeNamespace "{name : 'namespace1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 NVMeNamespaceStats "{namespace_id : {value : 'namespace1'} }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNVMfRemoteController "{nv_mf_remote_controller : {id: {value : 'NvmeTcp12'}, traddr:'11.11.11.2', subnqn:'nqn.2016-06.com.opi.spdk.target0', trsvcid:'4444', trtype:'NVME_TRANSPORT_TCP', adrfam:'NVMF_ADRFAM_IPV4', hostnqn:'nqn.2014-08.org.nvmexpress:uuid:feb98abe-d51f-40c8-b348-2753f3571d3c'}}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNVMfRemoteControllers "{}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNVMfRemoteController "{name: 'NvmeTcp12'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNVMfRemoteController "{name: 'NvmeTcp12'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNVMeNamespace "{name : 'namespace1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNVMeController "{name : 'controller1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNVMeSubsystem "{name : 'subsystem2'}"
```
