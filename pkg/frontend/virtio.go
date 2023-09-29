// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"context"
	"fmt"
	"log"
	"path"
	"sort"

	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-nvidia-bridge/pkg/models"
	"github.com/opiproject/opi-spdk-bridge/pkg/utils"

	"github.com/google/uuid"
	"go.einride.tech/aip/fieldbehavior"
	"go.einride.tech/aip/fieldmask"
	"go.einride.tech/aip/resourceid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func sortVirtioBlks(virtioBlks []*pb.VirtioBlk) {
	sort.Slice(virtioBlks, func(i int, j int) bool {
		return virtioBlks[i].Name < virtioBlks[j].Name
	})
}

// CreateVirtioBlk creates a Virtio block device
func (s *Server) CreateVirtioBlk(_ context.Context, in *pb.CreateVirtioBlkRequest) (*pb.VirtioBlk, error) {
	// check input correctness
	if err := s.validateCreateVirtioBlkRequest(in); err != nil {
		return nil, err
	}
	// see https://google.aip.dev/133#user-specified-ids
	resourceID := resourceid.NewSystemGenerated()
	if in.VirtioBlkId != "" {
		log.Printf("client provided the ID of a resource %v, ignoring the name field %v", in.VirtioBlkId, in.VirtioBlk.Name)
		resourceID = in.VirtioBlkId
	}
	in.VirtioBlk.Name = utils.ResourceIDToVolumeName(resourceID)
	// idempotent API when called with same key, should return same object
	controller, ok := s.VirtioCtrls[in.VirtioBlk.Name]
	if ok {
		log.Printf("Already existing NvmeController with id %v", in.VirtioBlk.Name)
		return controller, nil
	}
	// not found, so create a new one
	params := models.NvdaControllerVirtioBlkCreateParams{
		Serial: resourceID,
		Bdev:   in.VirtioBlk.VolumeNameRef,
		PfID:   int(in.VirtioBlk.PcieId.PhysicalFunction.Value),
		// VfID:             int(in.VirtioBlk.PcieId.VirtualFunction),
		NumQueues:        int(in.VirtioBlk.MaxIoQps),
		BdevType:         "spdk",
		EmulationManager: "mlx5_0",
	}
	var result models.NvdaControllerVirtioBlkCreateResult
	err := s.rpc.Call("controller_virtio_blk_create", &params, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if result == "" {
		msg := fmt.Sprintf("Could not create virtio-blk: %s", resourceID)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	response := utils.ProtoClone(in.VirtioBlk)
	// response.Status = &pb.NvmeControllerStatus{Active: true}
	s.VirtioCtrls[in.VirtioBlk.Name] = response
	return response, nil
}

// DeleteVirtioBlk deletes a Virtio block device
func (s *Server) DeleteVirtioBlk(_ context.Context, in *pb.DeleteVirtioBlkRequest) (*emptypb.Empty, error) {
	// check input correctness
	if err := s.validateDeleteVirtioBlkRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	controller, ok := s.VirtioCtrls[in.Name]
	if !ok {
		if in.AllowMissing {
			return &emptypb.Empty{}, nil
		}
		return nil, fmt.Errorf("error finding controller %s", in.Name)
	}
	params := models.NvdaControllerVirtioBlkDeleteParams{
		Name:  in.Name,
		Force: true,
	}
	var result models.NvdaControllerVirtioBlkDeleteResult
	err := s.rpc.Call("controller_virtio_blk_delete", &params, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		log.Printf("Could not delete: %v", in)
	}
	delete(s.VirtioCtrls, controller.Name)
	return &emptypb.Empty{}, nil
}

// UpdateVirtioBlk updates a Virtio block device
func (s *Server) UpdateVirtioBlk(_ context.Context, in *pb.UpdateVirtioBlkRequest) (*pb.VirtioBlk, error) {
	// check input correctness
	if err := s.validateUpdateVirtioBlkRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	volume, ok := s.VirtioCtrls[in.VirtioBlk.Name]
	if !ok {
		if in.AllowMissing {
			log.Printf("TODO: in case of AllowMissing, create a new resource, don;t return error")
		}
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.VirtioBlk.Name)
		return nil, err
	}
	resourceID := path.Base(volume.Name)
	// update_mask = 2
	if err := fieldmask.Validate(in.UpdateMask, in.VirtioBlk); err != nil {
		return nil, err
	}
	log.Printf("TODO: use resourceID=%v", resourceID)
	return nil, status.Errorf(codes.Unimplemented, "UpdateVirtioBlk method is not implemented")
}

// ListVirtioBlks lists Virtio block devices
func (s *Server) ListVirtioBlks(_ context.Context, in *pb.ListVirtioBlksRequest) (*pb.ListVirtioBlksResponse, error) {
	if err := fieldbehavior.ValidateRequiredFields(in); err != nil {
		return nil, err
	}

	size, offset, perr := utils.ExtractPagination(in.PageSize, in.PageToken, s.Pagination)
	if perr != nil {
		return nil, perr
	}
	var result []models.NvdaControllerListResult
	err := s.rpc.Call("controller_list", nil, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	token, hasMoreElements := "", false
	log.Printf("Limiting result len(%d) to [%d:%d]", len(result), offset, size)
	result, hasMoreElements = utils.LimitPagination(result, offset, size)
	if hasMoreElements {
		token = uuid.New().String()
		s.Pagination[token] = offset + size
	}
	Blobarray := []*pb.VirtioBlk{}
	for i := range result {
		r := &result[i]
		if r.Type == "virtio_blk" {
			ctrl := &pb.VirtioBlk{
				Name: utils.ResourceIDToVolumeName(r.Name),
				PcieId: &pb.PciEndpoint{
					PhysicalFunction: wrapperspb.Int32(int32(r.PciIndex)),
					VirtualFunction:  wrapperspb.Int32(0),
					PortId:           wrapperspb.Int32(0),
				},
				VolumeNameRef: "TBD"}
			Blobarray = append(Blobarray, ctrl)
		}
	}
	sortVirtioBlks(Blobarray)
	return &pb.ListVirtioBlksResponse{VirtioBlks: Blobarray}, nil
}

// GetVirtioBlk gets a Virtio block device
func (s *Server) GetVirtioBlk(_ context.Context, in *pb.GetVirtioBlkRequest) (*pb.VirtioBlk, error) {
	// check input correctness
	if err := s.validateGetVirtioBlkRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	_, ok := s.VirtioCtrls[in.Name]
	if !ok {
		msg := fmt.Sprintf("Could not find Controller: %s", in.Name)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	var result []models.NvdaControllerListResult
	err := s.rpc.Call("controller_list", nil, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	resourceID := path.Base(in.Name)
	for i := range result {
		r := &result[i]
		if r.Name == resourceID && r.Type == "virtio_blk" {
			return &pb.VirtioBlk{
				Name: utils.ResourceIDToVolumeName(r.Name),
				PcieId: &pb.PciEndpoint{
					PhysicalFunction: wrapperspb.Int32(int32(r.PciIndex)),
					VirtualFunction:  wrapperspb.Int32(0),
					PortId:           wrapperspb.Int32(0),
				},
				VolumeNameRef: "TBD"}, nil
		}
	}
	msg := fmt.Sprintf("Could not find Controller: %s", in.Name)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

// StatsVirtioBlk gets a Virtio block device stats
func (s *Server) StatsVirtioBlk(_ context.Context, in *pb.StatsVirtioBlkRequest) (*pb.StatsVirtioBlkResponse, error) {
	// check input correctness
	if err := s.validateStatsVirtioBlkRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	var result models.NvdaControllerNvmeStatsResult
	err := s.rpc.Call("controller_virtio_blk_get_iostat", nil, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for _, c := range result.Controllers {
		for _, r := range c.Bdevs {
			if r.BdevName == in.Name {
				return &pb.StatsVirtioBlkResponse{Stats: &pb.VolumeStats{
					ReadOpsCount:  int32(r.ReadIos),
					WriteOpsCount: int32(r.WriteIos),
				}}, nil
			}
		}
	}
	msg := fmt.Sprintf("Could not find Controller: %s", in.Name)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}
