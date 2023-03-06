// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"context"
	"fmt"
	"log"

	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	spdk "github.com/opiproject/opi-spdk-bridge/pkg/models"

	"github.com/ulule/deepcopier"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	errFailedSpdkCall           = status.Error(codes.Unknown, "Failed to execute SPDK call")
	errUnexpectedSpdkCallResult = status.Error(codes.FailedPrecondition, "Unexpected SPDK call result.")
)

// CreateVirtioBlk creates a Virtio block device
func (s *Server) CreateVirtioBlk(ctx context.Context, in *pb.CreateVirtioBlkRequest) (*pb.VirtioBlk, error) {
	log.Printf("CreateVirtioBlk: Received from client: %v", in)
	params := spdk.VhostCreateBlkControllerParams{
		Ctrlr:   in.VirtioBlk.Id.Value,
		DevName: in.VirtioBlk.VolumeId.Value,
	}
	var result spdk.VhostCreateBlkControllerResult
	err := s.rpc.Call("vhost_create_blk_controller", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, fmt.Errorf("%w for %v", errFailedSpdkCall, in)
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		log.Printf("Could not create: %v", in)
		return nil, fmt.Errorf("%w for %v", errUnexpectedSpdkCallResult, in)
	}

	response := &pb.VirtioBlk{}
	err = deepcopier.Copy(in.VirtioBlk).To(response)
	if err != nil {
		log.Printf("Error at response creation: %v", err)
		return nil, status.Error(codes.Internal, "Failed to construct device create response")
	}
	return response, nil
}

// DeleteVirtioBlk deletes a Virtio block device
func (s *Server) DeleteVirtioBlk(ctx context.Context, in *pb.DeleteVirtioBlkRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteVirtioBlk: Received from client: %v", in)
	params := spdk.VhostDeleteControllerParams{
		Ctrlr: in.Name,
	}
	var result spdk.VhostDeleteControllerResult
	err := s.rpc.Call("vhost_delete_controller", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		log.Printf("Could not delete: %v", in)
	}
	return &emptypb.Empty{}, nil
}

// UpdateVirtioBlk updates a Virtio block device
func (s *Server) UpdateVirtioBlk(ctx context.Context, in *pb.UpdateVirtioBlkRequest) (*pb.VirtioBlk, error) {
	log.Printf("Received from client: %v", in)
	return &pb.VirtioBlk{}, nil
}

// ListVirtioBlks lists Virtio block devices
func (s *Server) ListVirtioBlks(ctx context.Context, in *pb.ListVirtioBlksRequest) (*pb.ListVirtioBlksResponse, error) {
	log.Printf("ListVirtioBlks: Received from client: %v", in)
	var result []spdk.VhostGetControllersResult
	err := s.rpc.Call("vhost_get_controllers", nil, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	Blobarray := make([]*pb.VirtioBlk, len(result))
	for i := range result {
		r := &result[i]
		Blobarray[i] = &pb.VirtioBlk{Id: &pc.ObjectKey{Value: r.Ctrlr}}
	}
	return &pb.ListVirtioBlksResponse{VirtioBlks: Blobarray}, nil
}

// GetVirtioBlk gets a Virtio block device
func (s *Server) GetVirtioBlk(ctx context.Context, in *pb.GetVirtioBlkRequest) (*pb.VirtioBlk, error) {
	log.Printf("GetVirtioBlk: Received from client: %v", in)
	params := spdk.VhostGetControllersParams{
		Name: in.Name,
	}
	var result []spdk.VhostGetControllersResult
	err := s.rpc.Call("vhost_get_controllers", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if len(result) != 1 {
		msg := fmt.Sprintf("expecting exactly 1 result, got %d", len(result))
		log.Print(msg)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	return &pb.VirtioBlk{Id: &pc.ObjectKey{Value: result[0].Ctrlr}}, nil
}

// VirtioBlkStats gets a Virtio block device stats
func (s *Server) VirtioBlkStats(ctx context.Context, in *pb.VirtioBlkStatsRequest) (*pb.VirtioBlkStatsResponse, error) {
	log.Printf("Received from client: %v", in)
	return &pb.VirtioBlkStatsResponse{}, nil
}
