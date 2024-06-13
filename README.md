# OPI gRPC to Nvidia SDK bridge third party repo

[![Linters](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/linters.yml/badge.svg)](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/linters.yml)
[![CodeQL](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/codeql.yml/badge.svg)](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/opiproject/opi-nvidia-bridge/badge)](https://securityscorecards.dev/viewer/?platform=github.com&org=opiproject&repo=opi-nvidia-bridge)
[![tests](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/go.yml/badge.svg)](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/go.yml)
[![Docker](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/opiproject/opi-nvidia-bridge/actions/workflows/docker-publish.yml)
[![License](https://img.shields.io/github/license/opiproject/opi-nvidia-bridge?style=flat-square&color=blue&label=License)](https://github.com/opiproject/opi-nvidia-bridge/blob/master/LICENSE)
[![codecov](https://codecov.io/gh/opiproject/opi-nvidia-bridge/branch/main/graph/badge.svg)](https://codecov.io/gh/opiproject/opi-nvidia-bridge)
[![Go Report Card](https://goreportcard.com/badge/github.com/opiproject/opi-nvidia-bridge)](https://goreportcard.com/report/github.com/opiproject/opi-nvidia-bridge)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/opiproject/opi-nvidia-bridge)
[![Pulls](https://img.shields.io/docker/pulls/opiproject/opi-nvidia-bridge.svg?logo=docker&style=flat&label=Pulls)](https://hub.docker.com/r/opiproject/opi-nvidia-bridge)
[![Last Release](https://img.shields.io/github/v/release/opiproject/opi-nvidia-bridge?label=Latest&style=flat-square&logo=go)](https://github.com/opiproject/opi-nvidia-bridge/releases)
[![GitHub stars](https://img.shields.io/github/stars/opiproject/opi-nvidia-bridge.svg?style=flat-square&label=github%20stars)](https://github.com/opiproject/opi-nvidia-bridge)
[![GitHub Contributors](https://img.shields.io/github/contributors/opiproject/opi-nvidia-bridge.svg?style=flat-square)](https://github.com/opiproject/opi-nvidia-bridge/graphs/contributors)

This is a Nvidia app (bridge) to OPI APIs for storage, inventory, ipsec and networking (future).

## I Want To Contribute

This project welcomes contributions and suggestions.  We are happy to have the Community involved via submission of **Issues and Pull Requests** (with substantive content or even just fixes). We are hoping for the documents, test framework, etc. to become a community process with active engagement.  PRs can be reviewed by by any number of people, and a maintainer may accept.

See [CONTRIBUTING](https://github.com/opiproject/opi/blob/main/CONTRIBUTING.md) and [GitHub Basic Process](https://github.com/opiproject/opi/blob/main/doc-github-rules.md) for more details.

## Documentation

* SNAP <https://docs.nvidia.com/networking/display/bluefield3snaplatest/SNAP+RPC+Commands>
* Doca <https://docs.nvidia.com/doca/sdk/emulated-devices/index.html>
* and <https://docs.nvidia.com/networking/display/BlueFieldDPUOSLatest/BlueField+SNAP+on+DPU>

## Getting started

build like this:

```bash
go build -v -o /opi-nvidia-bridge ./cmd/...
```

import like this:

```go
import "github.com/opiproject/opi-nvidia-bridge/pkg/frontend"
```

## FW config

for Nvme:

```bash
sudo mlxconfig -d /dev/mst/mt41686_pciconf0 s NVME_EMULATION_ENABLE=1
sudo mlxconfig -d /dev/mst/mt41686_pciconf0 s NVME_EMULATION_NUM_PF=2 NVME_EMULATION_NUM_VF=2
```

for VirtioBlk:

```bash
sudo mlxconfig -d /dev/mst/mt41686_pciconf0 s VIRTIO_BLK_EMULATION_ENABLE=1
sudo mlxconfig -d /dev/mst/mt41686_pciconf0 s VIRTIO_BLK_EMULATION_NUM_PF=2 VIRTIO_BLK_EMULATION_NUM_VF=2
```

And then power cycle your system.

## Snap Service

start from systemd:

```bash
sudo systemctl start mlnx_snap
```

or from container, see [documentation](https://docs.nvidia.com/networking/display/bluefield3snaplatest/snap+deployment)

Make sure `/var/tmp/spdk.sock` is created. OPI bridge is using it to communicate with SNAP service.

## Using docker

Before initiating the bridge, the [Redis](https://redis.io/) and [Jaeger](https://www.jaegertracing.io/) services must be operational. To specify non-standard ports for these services, use the `--help` command with the binary to find out which parameters needs to be passed.

on DPU/IPU (i.e. with IP=10.10.10.1) run

```bash
$ docker run --rm -it -v /var/tmp/:/var/tmp/ -p 50051:50051 ghcr.io/opiproject/opi-nvidia-bridge:main
2023/09/12 20:29:05 Connection to SPDK will be via: unix detected from /var/tmp/spdk.sock
2023/09/12 20:29:05 gRPC server listening at [::]:50051
2023/09/12 20:29:05 HTTP Server listening at 8082
```

on X86 management VM run

reflection

```bash
$ docker run --network=host --rm -it namely/grpc-cli ls --json_input --json_output localhost:50051
grpc.reflection.v1alpha.ServerReflection
opi_api.inventory.v1.InventorySvc
opi_api.security.v1.IPsec
opi_api.storage.v1.AioVolumeService
opi_api.storage.v1.FrontendNvmeService
opi_api.storage.v1.FrontendVirtioBlkService
opi_api.storage.v1.FrontendVirtioScsiService
opi_api.storage.v1.MiddleendService
opi_api.storage.v1.NvmeRemoteControllerService
opi_api.storage.v1.NullVolumeService
```

full test suite

```bash
docker run --rm -it --network=host docker.io/opiproject/godpu:main inventory get --addr="10.10.10.10:50051"
docker run --rm -it --network=host docker.io/opiproject/godpu:main storage test --addr="10.10.10.10:50051"
docker run --rm -it --network=host docker.io/opiproject/godpu:main ipsec test --addr=10.10.10.10:50151 --pingaddr=8.8.8.1"
```

run either gRPC or HTTP requests

```bash
# gRPC requests
docker run --network=host --rm -it namely/grpc-cli ls   --json_input --json_output 10.10.10.10:50051 -l
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNvmeSubsystem "{nvme_subsystem : {spec : {nqn: 'nqn.2022-09.io.spdk:opitest2', serial_number: 'myserial2', model_number: 'mymodel2', max_namespaces: 11} }, nvme_subsystem_id : 'subsystem2' }"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNvmeSubsystems "{}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNvmeSubsystem "{name : 'nvmeSubsystems/subsystem2'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNvmeController "{parent: 'nvmeSubsystems/subsystem2', nvme_controller : {spec : {nvme_controller_id: 2, pcie_id : {physical_function : 0, virtual_function : 0, port_id: 0}, max_nsq:5, max_ncq:5, 'trtype': 'NVME_TRANSPORT_TYPE_PCIE' } }, nvme_controller_id : 'controller1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNvmeControllers "{parent : 'nvmeSubsystems/subsystem2'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNvmeController "{name : 'nvmeSubsystems/subsystem2/nvmeControllers/controller1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNvmeNamespace "{parent: 'nvmeSubsystems/subsystem2', nvme_namespace : {spec : {volume_name_ref : 'Malloc0', 'host_nsid' : '10', uuid:{value : '1b4e28ba-2fa1-11d2-883f-b9a761bde3fb'}, nguid: '1b4e28ba-2fa1-11d2-883f-b9a761bde3fb', eui64: 1967554867335598546 } }, nvme_namespace_id: 'namespace1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNvmeNamespaces "{parent : 'nvmeSubsystems/subsystem2'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNvmeNamespace "{name : 'nvmeSubsystems/subsystem2/nvmeNamespaces/namespace1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 StatsNvmeNamespace "{name : 'nvmeSubsystems/subsystem2/nvmeNamespaces/namespace1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNvmeRemoteController "{nvme_remote_controller : {multipath: 'NVME_MULTIPATH_MULTIPATH'}, nvme_remote_controller_id: 'nvmetcp12'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNvmeRemoteControllers "{}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNvmeRemoteController "{name: 'nvmeRemoteControllers/nvmetcp12'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNvmePath "{parent: 'nvmeRemoteControllers/nvmetcp12', nvme_path: {traddr:'11.11.11.2', trtype:'NVME_TRANSPORT_TYPE_TCP', fabrics: {adrfam:'NVME_ADDRESS_FAMILY_IPV4', subnqn:'nqn.2016-06.com.opi.spdk.target0', trsvcid:'4444', hostnqn:'nqn.2014-08.org.nvmexpress:uuid:feb98abe-d51f-40c8-b348-2753f3571d3c'}}, nvme_path_id: 'nvmetcp12path0'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNvmeRemoteController "{nvme_remote_controller : {multipath: 'NVME_MULTIPATH_DISABLE'}, nvme_remote_controller_id: 'nvmepcie13'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 CreateNvmePath "{parent: 'nvmeRemoteControllers/nvmepcie13', nvme_path : {traddr:'0000:01:00.0', trtype:'NVME_TRANSPORT_TYPE_PCIE'}, nvme_path_id: 'nvmepcie13path0'}"

docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 ListNvmePaths "{parent : 'nvmeRemoteControllers/nvmepcie13'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNvmePath "{name: 'nvmeRemoteControllers/nvmepcie13/nvmePaths/nvmepcie13path0'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNvmeRemoteController "{name: 'volumes/nvmepcie13'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 GetNvmePath "{name: 'nvmeRemoteControllers/nvmepcie13'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNvmePath "{name: 'nvmeRemoteControllers/nvmetcp12/nvmePaths/nvmetcp12path0'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNvmeRemoteController "{name: 'nvmeRemoteControllers/nvmetcp12'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNvmeNamespace "{name : 'nvmeSubsystems/subsystem2/nvmeNamespaces/namespace1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNvmeController "{name : 'nvmeSubsystems/subsystem2/nvmeControllers/controller1'}"
docker run --network=host --rm -it namely/grpc-cli call --json_input --json_output 10.10.10.10:50051 DeleteNvmeSubsystem "{name : 'nvmeSubsystems/subsystem2'}"
```

```bash
# HTTP requests
# inventory
curl -kL http://10.10.10.10:8082/v1/inventory/1/inventory/2
# Nvme
# create
curl -X POST -f http://10.10.10.10:8082/v1/nvmeRemoteControllers?nvme_remote_controller_id=nvmetcp12 -d '{"multipath": "NVME_MULTIPATH_MULTIPATH"}'
curl -X POST -f http://10.10.10.10:8082/v1/nvmeRemoteControllers/nvmetcp12/nvmePaths?nvme_path_id=nvmetcp12path0 -d '{"traddr":"11.11.11.2", "trtype":"NVME_TRANSPORT_TYPE_TCP", "fabrics":{"subnqn":"nqn.2016-06.com.opi.spdk.target0", "trsvcid":"4444", "adrfam":"NVME_ADDRESS_FAMILY_IPV4", "hostnqn":"nqn.2014-08.org.nvmexpress:uuid:feb98abe-d51f-40c8-b348-2753f3571d3c"}}'
curl -X POST -f http://10.10.10.10:8082/v1/nvmeSubsystems?nvme_subsystem_id=subsys0 -d '{"spec": {"nqn": "nqn.2022-09.io.spdk:opitest1"}}'
curl -X POST -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeNamespaces?nvme_namespace_id=namespace0 -d '{"spec": {"volume_name_ref": "Malloc0", "host_nsid": 10}}'
curl -X POST -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeControllers?nvme_controller_id=ctrl0 -d '{"spec": {"trtype": "NVME_TRANSPORT_TYPE_TCP", "fabrics_id":{"traddr": "127.0.0.1", "trsvcid": "4421", "adrfam": "NVME_ADDRESS_FAMILY_IPV4"}}}'
# get
curl -X GET -f http://10.10.10.10:8082/v1/nvmeRemoteControllers/nvmetcp12
curl -X GET -f http://10.10.10.10:8082/v1/nvmeRemoteControllers/nvmetcp12/nvmePaths/nvmetcp12path0
curl -X GET -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0
curl -X GET -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeNamespaces/namespace0
curl -X GET -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeControllers/ctrl0
# list
curl -X GET -f http://10.10.10.10:8082/v1/nvmeRemoteControllers
curl -X GET -f http://10.10.10.10:8082/v1/nvmeRemoteControllers/nvmetcp12/nvmePaths
curl -X GET -f http://10.10.10.10:8082/v1/nvmeSubsystems
curl -X GET -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeNamespaces
curl -X GET -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeControllers
# stats
curl -X GET -f http://10.10.10.10:8082/v1/nvmeRemoteControllers/nvmetcp12:stats
curl -X GET -f http://10.10.10.10:8082/v1/nvmeRemoteControllers/nvmetcp12/nvmePaths/nvmetcp12path0:stats
curl -X GET -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0:stats
curl -X GET -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeNamespaces/namespace0:stats
curl -X GET -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeControllers/ctrl0:stats
# update
curl -X PATCH -f http://10.10.10.10:8082/v1/nvmeRemoteControllers/nvmetcp12 -d '{"multipath": "NVME_MULTIPATH_MULTIPATH"}'
curl -X PATCH -f http://10.10.10.10:8082/v1/nvmeRemoteControllers/nvmetcp12/nvmePaths/nvmetcp12path0 -d '{"traddr":"11.11.11.2", "trtype":"NVME_TRANSPORT_TYPE_TCP", "fabrics":{"subnqn":"nqn.2016-06.com.opi.spdk.target0", "trsvcid":"4444", "adrfam":"NVME_ADDRESS_FAMILY_IPV4", "hostnqn":"nqn.2014-08.org.nvmexpress:uuid:feb98abe-d51f-40c8-b348-2753f3571d3c"}}'
curl -X PATCH -k http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeNamespaces/namespace0 -d '{"spec": {"volume_name_ref": "Malloc1", "host_nsid": 10}}'
curl -X PATCH -k http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeControllers/ctrl0 -d '{"spec": {"trtype": "NVME_TRANSPORT_TYPE_TCP", "fabrics_id":{"traddr": "127.0.0.1", "trsvcid": "4421", "adrfam": "NVME_ADDRESS_FAMILY_IPV4"}}}'
# delete
curl -X DELETE -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeControllers/ctrl0
curl -X DELETE -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0/nvmeNamespaces/namespace0
curl -X DELETE -f http://10.10.10.10:8082/v1/nvmeSubsystems/subsys0
curl -X DELETE -f http://10.10.10.10:8082/v1/nvmeRemoteControllers/nvmetcp12/nvmePaths/nvmetcp12path0
curl -X DELETE -f http://10.10.10.10:8082/v1/nvmeRemoteControllers/nvmetcp12
```
