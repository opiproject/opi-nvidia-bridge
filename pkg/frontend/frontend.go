// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// Copyright (C) 2023 Intel Corporation

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/server"
)

// Server contains frontend related OPI services
type Server struct {
	pb.UnimplementedFrontendNvmeServiceServer
	Subsystems  map[string]*pb.NVMeSubsystem
	Controllers map[string]*pb.NVMeController
	Namespaces  map[string]*pb.NVMeNamespace
	VirtioCtrls map[string]*pb.VirtioBlk
	rpc         server.JSONRPC
}

// NewServer creates initialized instance of NVMe server
func NewServer(jsonRPC server.JSONRPC) *Server {
	return &Server{
		Subsystems:  make(map[string]*pb.NVMeSubsystem),
		Controllers: make(map[string]*pb.NVMeController),
		Namespaces:  make(map[string]*pb.NVMeNamespace),
		VirtioCtrls: make(map[string]*pb.VirtioBlk),
		rpc:         jsonRPC,
	}
}
