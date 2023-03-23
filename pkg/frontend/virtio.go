// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"context"
	"fmt"
	"log"

	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-nvidia-bridge/pkg/models"

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
func (s *Server) CreateVirtioBlk(_ context.Context, in *pb.CreateVirtioBlkRequest) (*pb.VirtioBlk, error) {
	log.Printf("CreateVirtioBlk: Received from client: %v", in)
	// idempotent API when called with same key, should return same object
	controller, ok := s.VirtioCtrls[in.VirtioBlk.Id.Value]
	if ok {
		log.Printf("Already existing NVMeController with id %v", in.VirtioBlk.Id.Value)
		return controller, nil
	}
	// not found, so create a new one
	params := models.NvdaControllerVirtioBlkCreateParams{
		Serial: in.VirtioBlk.Id.Value,
		Bdev:   in.VirtioBlk.VolumeId.Value,
		PfID:   int(in.VirtioBlk.PcieId.PhysicalFunction),
		// VfID:             int(in.VirtioBlk.PcieId.VirtualFunction),
		NumQueues:        int(in.VirtioBlk.MaxIoQps),
		BdevType:         "spdk",
		EmulationManager: "mlx5_0",
	}
	var result models.NvdaControllerVirtioBlkCreateResult
	err := s.rpc.Call("controller_virtio_blk_create", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, fmt.Errorf("%w for %v", errFailedSpdkCall, in)
	}
	log.Printf("Received from SPDK: %v", result)
	if result == "" {
		log.Printf("Could not create: %v", in)
		return nil, fmt.Errorf("%w for %v", errUnexpectedSpdkCallResult, in)
	}
	s.VirtioCtrls[in.VirtioBlk.Id.Value] = in.VirtioBlk
	// s.VirtioCtrls[in.VirtioBlk.Id.Value].Status = &pb.NVMeControllerStatus{Active: true}
	response := &pb.VirtioBlk{}
	err = deepcopier.Copy(in.VirtioBlk).To(response)
	if err != nil {
		log.Printf("Error at response creation: %v", err)
		return nil, status.Error(codes.Internal, "Failed to construct device create response")
	}
	return response, nil
}

// DeleteVirtioBlk deletes a Virtio block device
func (s *Server) DeleteVirtioBlk(_ context.Context, in *pb.DeleteVirtioBlkRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteVirtioBlk: Received from client: %v", in)
	controller, ok := s.VirtioCtrls[in.Name]
	if !ok {
		return nil, fmt.Errorf("error finding controller %s", in.Name)
	}
	params := models.NvdaControllerVirtioBlkDeleteParams{
		Name:  in.Name,
		Force: true,
	}
	var result models.NvdaControllerVirtioBlkDeleteResult
	err := s.rpc.Call("controller_virtio_blk_delete", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		log.Printf("Could not delete: %v", in)
	}
	delete(s.VirtioCtrls, controller.Id.Value)
	return &emptypb.Empty{}, nil
}

// UpdateVirtioBlk updates a Virtio block device
func (s *Server) UpdateVirtioBlk(_ context.Context, in *pb.UpdateVirtioBlkRequest) (*pb.VirtioBlk, error) {
	log.Printf("UpdateVirtioBlk: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateVirtioBlk method is not implemented")
}

// ListVirtioBlks lists Virtio block devices
func (s *Server) ListVirtioBlks(_ context.Context, in *pb.ListVirtioBlksRequest) (*pb.ListVirtioBlksResponse, error) {
	log.Printf("ListVirtioBlks: Received from client: %v", in)
	var result []models.NvdaControllerListResult
	err := s.rpc.Call("controller_list", nil, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	Blobarray := make([]*pb.VirtioBlk, len(result))
	for i := range result {
		r := &result[i]
		if r.Type == "virtio_blk" {
			Blobarray[i] = &pb.VirtioBlk{
				Id:       &pc.ObjectKey{Value: r.Name},
				PcieId:   &pb.PciEndpoint{PhysicalFunction: int32(r.PciIndex)},
				VolumeId: &pc.ObjectKey{Value: "TBD"}}
		}
	}
	return &pb.ListVirtioBlksResponse{VirtioBlks: Blobarray}, nil
}

// GetVirtioBlk gets a Virtio block device
func (s *Server) GetVirtioBlk(_ context.Context, in *pb.GetVirtioBlkRequest) (*pb.VirtioBlk, error) {
	log.Printf("GetVirtioBlk: Received from client: %v", in)
	_, ok := s.VirtioCtrls[in.Name]
	if !ok {
		msg := fmt.Sprintf("Could not find Controller: %s", in.Name)
		log.Print(msg)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	var result []models.NvdaControllerListResult
	err := s.rpc.Call("controller_list", nil, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for i := range result {
		r := &result[i]
		if r.Name == in.Name && r.Type == "virtio_blk" {
			return &pb.VirtioBlk{
				Id:       &pc.ObjectKey{Value: r.Name},
				PcieId:   &pb.PciEndpoint{PhysicalFunction: int32(r.PciIndex)},
				VolumeId: &pc.ObjectKey{Value: "TBD"}}, nil
		}
	}
	msg := fmt.Sprintf("Could not find Controller: %s", in.Name)
	log.Print(msg)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

// VirtioBlkStats gets a Virtio block device stats
func (s *Server) VirtioBlkStats(_ context.Context, in *pb.VirtioBlkStatsRequest) (*pb.VirtioBlkStatsResponse, error) {
	log.Printf("VirtioBlkStats: Received from client: %v", in)
	var result models.NvdaControllerNvmeStatsResult
	err := s.rpc.Call("controller_virtio_blk_get_iostat", nil, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for _, c := range result.Controllers {
		for _, r := range c.Bdevs {
			if r.BdevName == in.ControllerId.Value {
				return &pb.VirtioBlkStatsResponse{Id: in.ControllerId, Stats: &pb.VolumeStats{
					ReadOpsCount:  int32(r.ReadIos),
					WriteOpsCount: int32(r.WriteIos),
				}}, nil
			}
		}
	}
	msg := fmt.Sprintf("Could not find Controller: %s", in.ControllerId.Value)
	log.Print(msg)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}
