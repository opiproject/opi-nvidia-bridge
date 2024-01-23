// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// Copyright (C) 2023 Intel Corporation

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"log"

	"github.com/philippgille/gokv"

	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
)

// Server contains frontend related OPI services
type Server struct {
	pb.UnimplementedFrontendNvmeServiceServer
	pb.UnimplementedFrontendVirtioBlkServiceServer
	VirtioCtrls map[string]*pb.VirtioBlk
	NQNs        map[string]bool
	Pagination  map[string]int
	store       gokv.Store
	rpc         spdk.JSONRPC
}

// NewServer creates initialized instance of Nvme server
func NewServer(jsonRPC spdk.JSONRPC, store gokv.Store) *Server {
	if jsonRPC == nil {
		log.Panic("nil for JSONRPC is not allowed")
	}
	if store == nil {
		log.Panic("nil for Store is not allowed")
	}
	return &Server{
		VirtioCtrls: make(map[string]*pb.VirtioBlk),
		NQNs:        make(map[string]bool),
		Pagination:  make(map[string]int),
		store:       store,
		rpc:         jsonRPC,
	}
}
