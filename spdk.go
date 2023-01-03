// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.

package main

type NvdaSubsystemNvmeCreateParams struct {
	Nqn          string `json:"nqn"`
	SerialNumber string `json:"serial_number"`
	ModelNumber  string `json:"model_number"`
}

type NvdaSubsystemNvmeCreateResult bool

type NvdaSubsystemNvmeDeleteParams struct {
	Nqn string `json:"nqn"`
}

type NvdaSubsystemNvmeDeleteResult bool

// NvdaSubsystemNvmeListParams is empty

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

type NvdaControllerNvmeCreateParams struct {
	Nqn              string `json:"nqn"`
	EmulationManager string `json:"emulation_manager"`
	PfID             int    `json:"pf_id"`
	VfID             int    `json:"vf_id,omitempty"`
	NrIoQueues       int    `json:"nr_io_queues,omitempty"`
	MaxNamespaces    int    `json:"max_namespaces,omitempty"`
}

type NvdaControllerNvmeCreateResult struct {
	Name   string `json:"name"`
	Cntlid int    `json:"cntlid"`
}

type NvdaControllerNvmeDeleteParams struct {
	Subnqn string `json:"subnqn"`
	Cntlid int    `json:"cntlid"`
}

type NvdaControllerNvmeDeleteResult bool

// NvdaControllerNvmeListParams is empty

type NvdaControllerNvmeListResult struct {
	Subnqn           string `json:"subnqn"`
	Cntlid           int    `json:"cntlid"`
	Name             string `json:"name"`
	EmulationManager string `json:"emulation_manager"`
	Type             string `json:"type"`
	PciIndex         int    `json:"pci_index"`
	PciBdf           string `json:"pci_bdf"`
}

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

type NvdaControllerNvmeNamespaceAttachResult bool

type NvdaControllerNvmeNamespaceDetachParams struct {
	Nsid   int    `json:"nsid"`
	Subnqn string `json:"subnqn"`
	Cntlid int    `json:"cntlid"`
}

type NvdaControllerNvmeNamespaceDetachResult bool

type NvdaControllerNvmeNamespaceListParams struct {
	Subnqn string `json:"subnqn"`
	Cntlid int    `json:"cntlid"`
}

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

type GetVersionResult struct {
	Version string `json:"version"`
	Fields  struct {
		Major  int    `json:"major"`
		Minor  int    `json:"minor"`
		Patch  int    `json:"patch"`
		Suffix string `json:"suffix"`
	} `json:"fields"`
}
