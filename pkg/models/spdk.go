// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.

// Package models holds definitions for SPDK json RPC structs
package models

// NvdaSubsystemNvmeCreateParams represents a Nvidia subsystem create request
type NvdaSubsystemNvmeCreateParams struct {
	Nqn          string `json:"nqn"`
	SerialNumber string `json:"serial_number"`
	ModelNumber  string `json:"model_number"`
}

// NvdaSubsystemNvmeCreateResult represents a Nvidia subsystem create result
type NvdaSubsystemNvmeCreateResult bool

// NvdaSubsystemNvmeDeleteParams represents a Nvidia subsystem delete request
type NvdaSubsystemNvmeDeleteParams struct {
	Nqn string `json:"nqn"`
}

// NvdaSubsystemNvmeDeleteResult represents a Nvidia subsystem delete result
type NvdaSubsystemNvmeDeleteResult bool

// NvdaSubsystemNvmeListParams is empty

// NvdaSubsystemNvmeListResult represents a Nvidia subsystem list request
type NvdaSubsystemNvmeListResult struct {
	Nqn          string `json:"nqn"`
	SerialNumber string `json:"serial_number"`
	ModelNumber  string `json:"model_number"`
	Controllers  []struct {
		Name     string `json:"name"`
		Cntlid   int    `json:"cntlid"`
		PciBdf   string `json:"pci_bdf"`
		PciIndex int    `json:"pci_index"`
	} `json:"controllers"`
}

// NvdaControllerNvmeCreateParams represents a Nvidia Controller create request
type NvdaControllerNvmeCreateParams struct {
	Nqn              string `json:"nqn"`
	EmulationManager string `json:"emulation_manager"`
	PfID             int    `json:"pf_id"`
	VfID             int    `json:"vf_id,omitempty"`
	NrIoQueues       int    `json:"nr_io_queues,omitempty"`
	MaxNamespaces    int    `json:"max_namespaces,omitempty"`
}

// NvdaControllerNvmeCreateResult represents a Nvidia Controller create result
type NvdaControllerNvmeCreateResult struct {
	Name   string `json:"name"`
	Cntlid int    `json:"cntlid"`
}

// NvdaControllerNvmeDeleteParams represents a Nvidia Controller delete request
type NvdaControllerNvmeDeleteParams struct {
	Subnqn string `json:"subnqn"`
	Cntlid int    `json:"cntlid"`
}

// NvdaControllerNvmeDeleteResult represents a Nvidia Controller delete result
type NvdaControllerNvmeDeleteResult bool

// NvdaControllerListParams is empty (both Nvme and VirtIo)

// NvdaControllerListResult represents a Nvidia Controller list request (both Nvme and VirtIo)
type NvdaControllerListResult struct {
	Subnqn           string `json:"subnqn"`
	Cntlid           int    `json:"cntlid"`
	Name             string `json:"name"`
	EmulationManager string `json:"emulation_manager"`
	Type             string `json:"type"`
	PciIndex         int    `json:"pci_index"`
	PciBdf           string `json:"pci_bdf"`
}

// NvdaControllerNvmeNamespaceAttachParams represents a Nvidia controller attach namespaces request
type NvdaControllerNvmeNamespaceAttachParams struct {
	BdevType string `json:"bdev_type"`
	Bdev     string `json:"bdev"`
	Nsid     int    `json:"nsid"`
	Subnqn   string `json:"subnqn"`
	Cntlid   int    `json:"cntlid"`
	UUID     string `json:"uuid"`
	Nguid    string `json:"nguid"`
	Eui64    string `json:"eui64"`
}

// NvdaControllerNvmeNamespaceAttachResult represents a Nvidia controller attach namespaces result
type NvdaControllerNvmeNamespaceAttachResult bool

// NvdaControllerNvmeNamespaceDetachParams represents a Nvidia controller detach namespaces request
type NvdaControllerNvmeNamespaceDetachParams struct {
	Nsid   int    `json:"nsid"`
	Subnqn string `json:"subnqn"`
	Cntlid int    `json:"cntlid"`
}

// NvdaControllerNvmeNamespaceDetachResult represents a Nvidia controller detach namespaces result
type NvdaControllerNvmeNamespaceDetachResult bool

// NvdaControllerNvmeNamespaceListParams represents a Nvidia controller list of namespaces request
type NvdaControllerNvmeNamespaceListParams struct {
	Subnqn string `json:"subnqn"`
	Cntlid int    `json:"cntlid"`
}

// NvdaControllerNvmeNamespaceListResult represents a Nvidia controller list of namespaces result
type NvdaControllerNvmeNamespaceListResult struct {
	Name       string `json:"name"`
	Cntlid     int    `json:"cntlid"`
	Namespaces []struct {
		Nsid     int    `json:"nsid"`
		Bdev     string `json:"bdev"`
		BdevType string `json:"bdev_type"`
		Qn       string `json:"qn"`
		Protocol string `json:"protocol"`
	} `json:"Namespaces"`
}

// NvdaControllerNvmeStatsResult represents a Nvidia controller get stats result
type NvdaControllerNvmeStatsResult struct {
	Controllers []struct {
		Name  string `json:"name"`
		Bdevs []struct {
			BdevName          string `json:"bdev_name"`
			ReadIos           int    `json:"read_ios"`
			CompletedReadIos  int    `json:"completed_read_ios"`
			WriteIos          int    `json:"write_ios"`
			CompletedWriteIos int    `json:"completed_write_ios"`
			FlushIos          int    `json:"flush_ios"`
			CompletedFlushIos int    `json:"completed_flush_ios"`
			ErrReadIos        int    `json:"err_read_ios"`
			ErrWriteIos       int    `json:"err_write_ios"`
			ErrFlushIos       int    `json:"err_flush_ios"`
		} `json:"bdevs"`
	} `json:"controllers"`
}

// NvdaControllerVirtioBlkCreateParams represents a Nvidia Controller create request
type NvdaControllerVirtioBlkCreateParams struct {
	EmulationManager string `json:"emulation_manager"`
	BdevType         string `json:"bdev_type"`
	PfID             int    `json:"pf_id"`
	VfID             int    `json:"vf_id"`
	NumQueues        int    `json:"num_queues"`
	Bdev             string `json:"bdev"`
	Serial           string `json:"serial"`
}

// NvdaControllerVirtioBlkCreateResult represents a Nvidia Controller create result
type NvdaControllerVirtioBlkCreateResult string

// NvdaControllerVirtioBlkDeleteParams represents a Nvidia Controller delete request
type NvdaControllerVirtioBlkDeleteParams struct {
	Name  string `json:"name"`
	Force bool   `json:"force"`
}

// NvdaControllerVirtioBlkDeleteResult represents a Nvidia Controller delete result
type NvdaControllerVirtioBlkDeleteResult bool
