// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// Copyright (C) 2023 Intel Corporation

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
	"github.com/opiproject/opi-spdk-bridge/pkg/server"

	"github.com/google/uuid"
	"go.einride.tech/aip/fieldmask"
	"go.einride.tech/aip/resourceid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func sortNvmeControllers(controllers []*pb.NvmeController) {
	sort.Slice(controllers, func(i int, j int) bool {
		return controllers[i].Spec.NvmeControllerId < controllers[j].Spec.NvmeControllerId
	})
}

// CreateNvmeController creates an Nvme controller
func (s *Server) CreateNvmeController(_ context.Context, in *pb.CreateNvmeControllerRequest) (*pb.NvmeController, error) {
	log.Printf("CreateNvmeController: Received from client: %v", in)
	// check input parameters validity
	if in.NvmeController.Spec == nil || in.NvmeController.Spec.SubsystemId == nil || in.NvmeController.Spec.SubsystemId.Value == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid input subsystem parameters")
	}
	// see https://google.aip.dev/133#user-specified-ids
	resourceID := resourceid.NewSystemGenerated()
	if in.NvmeControllerId != "" {
		err := resourceid.ValidateUserSettable(in.NvmeControllerId)
		if err != nil {
			log.Printf("error: %v", err)
			return nil, err
		}
		log.Printf("client provided the ID of a resource %v, ignoring the name field %v", in.NvmeControllerId, in.NvmeController.Name)
		resourceID = in.NvmeControllerId
	}
	in.NvmeController.Name = server.ResourceIDToVolumeName(resourceID)
	// idempotent API when called with same key, should return same object
	controller, ok := s.Controllers[in.NvmeController.Name]
	if ok {
		log.Printf("Already existing NvmeController with id %v", in.NvmeController.Name)
		return controller, nil
	}
	// not found, so create a new one
	subsys, ok := s.Subsystems[in.NvmeController.Spec.SubsystemId.Value]
	if !ok {
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.NvmeController.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}

	params := models.NvdaControllerNvmeCreateParams{
		Nqn:              subsys.Spec.Nqn,
		EmulationManager: "mlx5_0",
		PfID:             int(in.NvmeController.Spec.PcieId.PhysicalFunction),
		// VfID:             int(in.NvmeController.Spec.PcieId.VirtualFunction),
		// MaxNamespaces:    int(in.NvmeController.Spec.MaxNsq),
		// NrIoQueues:       int(in.NvmeController.Spec.MaxNcq),
	}
	var result models.NvdaControllerNvmeCreateResult
	err := s.rpc.Call("controller_nvme_create", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if result.Cntlid < 0 {
		msg := fmt.Sprintf("Could not create CTRL: %s", in.NvmeController.Name)
		log.Print(msg)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	response := server.ProtoClone(in.NvmeController)
	response.Spec.NvmeControllerId = int32(result.Cntlid)
	response.Status = &pb.NvmeControllerStatus{Active: true}
	s.Controllers[in.NvmeController.Name] = response
	return response, nil
}

// DeleteNvmeController deletes an Nvme controller
func (s *Server) DeleteNvmeController(_ context.Context, in *pb.DeleteNvmeControllerRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteNvmeController: Received from client: %v", in)
	controller, ok := s.Controllers[in.Name]
	if !ok {
		if in.AllowMissing {
			return &emptypb.Empty{}, nil
		}
		return nil, fmt.Errorf("error finding controller %s", in.Name)
	}
	subsys, ok := s.Subsystems[controller.Spec.SubsystemId.Value]
	if !ok {
		err := status.Errorf(codes.NotFound, "unable to find key %s", controller.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}

	params := models.NvdaControllerNvmeDeleteParams{
		Subnqn: subsys.Spec.Nqn,
		Cntlid: int(controller.Spec.NvmeControllerId),
	}
	var result models.NvdaControllerNvmeDeleteResult
	err := s.rpc.Call("controller_nvme_delete", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not delete NQN:ID %s:%d", subsys.Spec.Nqn, controller.Spec.NvmeControllerId)
		log.Print(msg)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	delete(s.Controllers, controller.Name)
	return &emptypb.Empty{}, nil
}

// UpdateNvmeController updates an Nvme controller
func (s *Server) UpdateNvmeController(_ context.Context, in *pb.UpdateNvmeControllerRequest) (*pb.NvmeController, error) {
	log.Printf("UpdateNvmeController: Received from client: %v", in)
	volume, ok := s.Controllers[in.NvmeController.Name]
	if !ok {
		if in.AllowMissing {
			log.Printf("TODO: in case of AllowMissing, create a new resource, don;t return error")
		}
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.NvmeController.Name)
		log.Printf("error: %v", err)
		return nil, err
	}
	resourceID := path.Base(volume.Name)
	// update_mask = 2
	if err := fieldmask.Validate(in.UpdateMask, in.NvmeController); err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("TODO: use resourceID=%v", resourceID)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNvmeController method is not implemented")
}

// ListNvmeControllers lists Nvme controllers
func (s *Server) ListNvmeControllers(_ context.Context, in *pb.ListNvmeControllersRequest) (*pb.ListNvmeControllersResponse, error) {
	log.Printf("ListNvmeControllers: Received from client: %v", in)
	size, offset, perr := server.ExtractPagination(in.PageSize, in.PageToken, s.Pagination)
	if perr != nil {
		log.Printf("error: %v", perr)
		return nil, perr
	}
	subsys, ok := s.Subsystems[in.Parent]
	if !ok {
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.Parent)
		log.Printf("error: %v", err)
		return nil, err
	}
	var result []models.NvdaControllerListResult
	err := s.rpc.Call("controller_list", nil, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	token, hasMoreElements := "", false
	log.Printf("Limiting result len(%d) to [%d:%d]", len(result), offset, size)
	result, hasMoreElements = server.LimitPagination(result, offset, size)
	if hasMoreElements {
		token = uuid.New().String()
		s.Pagination[token] = offset + size
	}
	Blobarray := make([]*pb.NvmeController, len(result))
	for i := range result {
		r := &result[i]
		if r.Subnqn == subsys.Spec.Nqn && r.Type == "nvme" {
			Blobarray[i] = &pb.NvmeController{Spec: &pb.NvmeControllerSpec{NvmeControllerId: int32(r.Cntlid)}}
		}
	}
	sortNvmeControllers(Blobarray)
	return &pb.ListNvmeControllersResponse{NvmeControllers: Blobarray}, nil
}

// GetNvmeController gets an Nvme controller
func (s *Server) GetNvmeController(_ context.Context, in *pb.GetNvmeControllerRequest) (*pb.NvmeController, error) {
	log.Printf("GetNvmeController: Received from client: %v", in)
	controller, ok := s.Controllers[in.Name]
	if !ok {
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.Name)
		log.Printf("error: %v", err)
		return nil, err
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
		if r.Cntlid == int(controller.Spec.NvmeControllerId) && r.Type == "nvme" {
			return &pb.NvmeController{Spec: &pb.NvmeControllerSpec{NvmeControllerId: int32(r.Cntlid)}, Status: &pb.NvmeControllerStatus{Active: true}}, nil
		}
	}
	msg := fmt.Sprintf("Could not find NvmeControllerId: %d", controller.Spec.NvmeControllerId)
	log.Print(msg)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

// NvmeControllerStats gets an Nvme controller stats
func (s *Server) NvmeControllerStats(_ context.Context, in *pb.NvmeControllerStatsRequest) (*pb.NvmeControllerStatsResponse, error) {
	log.Printf("NvmeControllerStats: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "NvmeControllerStats method is not implemented")
}
