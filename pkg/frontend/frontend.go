// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// Copyright (C) 2023 Intel Corporation

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
)

// Server contains frontend related OPI services
type Server struct {
	pb.UnimplementedFrontendNvmeServiceServer
	Subsystems  map[string]*pb.NvmeSubsystem
	Controllers map[string]*pb.NvmeController
	Namespaces  map[string]*pb.NvmeNamespace
	VirtioCtrls map[string]*pb.VirtioBlk
	Pagination  map[string]int
	rpc         spdk.JSONRPC
}

// NewServer creates initialized instance of Nvme server
func NewServer(jsonRPC spdk.JSONRPC) *Server {
	return &Server{
		Subsystems:  make(map[string]*pb.NvmeSubsystem),
		Controllers: make(map[string]*pb.NvmeController),
		Namespaces:  make(map[string]*pb.NvmeNamespace),
		VirtioCtrls: make(map[string]*pb.VirtioBlk),
		Pagination:  make(map[string]int),
		rpc:         jsonRPC,
	}
}
