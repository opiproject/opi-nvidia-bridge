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
	"strconv"

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
)

func sortNvmeNamespaces(namespaces []*pb.NvmeNamespace) {
	sort.Slice(namespaces, func(i int, j int) bool {
		return namespaces[i].Spec.HostNsid < namespaces[j].Spec.HostNsid
	})
}

// CreateNvmeNamespace creates an Nvme namespace
func (s *Server) CreateNvmeNamespace(ctx context.Context, in *pb.CreateNvmeNamespaceRequest) (*pb.NvmeNamespace, error) {
	// check input correctness
	if err := s.validateCreateNvmeNamespaceRequest(in); err != nil {
		return nil, err
	}
	// see https://google.aip.dev/133#user-specified-ids
	resourceID := resourceid.NewSystemGenerated()
	if in.NvmeNamespaceId != "" {
		log.Printf("client provided the ID of a resource %v, ignoring the name field %v", in.NvmeNamespaceId, in.NvmeNamespace.Name)
		resourceID = in.NvmeNamespaceId
	}
	in.NvmeNamespace.Name = utils.ResourceIDToNamespaceName(
		utils.GetSubsystemIDFromNvmeName(in.Parent), resourceID,
	)
	// idempotent API when called with same key, should return same object
	namespace, ok := s.Namespaces[in.NvmeNamespace.Name]
	if ok {
		log.Printf("Already existing NvmeNamespace with id %v", in.NvmeNamespace.Name)
		return namespace, nil
	}
	// not found, so create a new one
	subsys, ok := s.Subsystems[in.Parent]
	if !ok {
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.Parent)
		return nil, err
	}
	// TODO: do lookup through VolumeId key instead of using it's value
	params := models.NvdaControllerNvmeNamespaceAttachParams{
		BdevType: "spdk",
		Bdev:     in.NvmeNamespace.Spec.VolumeNameRef,
		Nsid:     int(in.NvmeNamespace.Spec.HostNsid),
		Subnqn:   subsys.Spec.Nqn,
		Cntlid:   0,
		UUID:     in.NvmeNamespace.Spec.Uuid.Value,
		Nguid:    in.NvmeNamespace.Spec.Nguid,
		Eui64:    strconv.FormatInt(in.NvmeNamespace.Spec.Eui64, 10),
	}
	var result models.NvdaControllerNvmeNamespaceAttachResult
	err := s.rpc.Call(ctx, "controller_nvme_namespace_attach", &params, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not create NS: %s", in.NvmeNamespace.Name)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	response := utils.ProtoClone(in.NvmeNamespace)
	response.Status = &pb.NvmeNamespaceStatus{
		State:     pb.NvmeNamespaceStatus_STATE_ENABLED,
		OperState: pb.NvmeNamespaceStatus_OPER_STATE_ONLINE,
	}
	s.Namespaces[in.NvmeNamespace.Name] = response
	return response, nil
}

// DeleteNvmeNamespace deletes an Nvme namespace
func (s *Server) DeleteNvmeNamespace(ctx context.Context, in *pb.DeleteNvmeNamespaceRequest) (*emptypb.Empty, error) {
	// check input correctness
	if err := s.validateDeleteNvmeNamespaceRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	namespace, ok := s.Namespaces[in.Name]
	if !ok {
		if in.AllowMissing {
			return &emptypb.Empty{}, nil
		}
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.Name)
		return nil, err
	}
	subsysName := utils.ResourceIDToSubsystemName(
		utils.GetSubsystemIDFromNvmeName(in.Name),
	)
	subsys, ok := s.Subsystems[subsysName]
	if !ok {
		err := fmt.Errorf("unable to find subsystem %s", subsysName)
		return nil, err
	}

	// TODO: fix hard-coded Cntlid
	params := models.NvdaControllerNvmeNamespaceDetachParams{
		Nsid:   int(namespace.Spec.HostNsid),
		Subnqn: subsys.Spec.Nqn,
		Cntlid: 0,
	}
	var result models.NvdaControllerNvmeNamespaceDetachResult
	err := s.rpc.Call(ctx, "controller_nvme_namespace_detach", &params, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not delete NS: %s", in.Name)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	delete(s.Namespaces, namespace.Name)
	return &emptypb.Empty{}, nil
}

// UpdateNvmeNamespace updates an Nvme namespace
func (s *Server) UpdateNvmeNamespace(_ context.Context, in *pb.UpdateNvmeNamespaceRequest) (*pb.NvmeNamespace, error) {
	// check input correctness
	if err := s.validateUpdateNvmeNamespaceRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	volume, ok := s.Namespaces[in.NvmeNamespace.Name]
	if !ok {
		if in.AllowMissing {
			log.Printf("TODO: in case of AllowMissing, create a new resource, don;t return error")
		}
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.NvmeNamespace.Name)
		return nil, err
	}
	resourceID := path.Base(volume.Name)
	// update_mask = 2
	if err := fieldmask.Validate(in.UpdateMask, in.NvmeNamespace); err != nil {
		return nil, err
	}
	log.Printf("TODO: use resourceID=%v", resourceID)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNvmeNamespace method is not implemented")
}

// ListNvmeNamespaces lists Nvme namespaces
func (s *Server) ListNvmeNamespaces(ctx context.Context, in *pb.ListNvmeNamespacesRequest) (*pb.ListNvmeNamespacesResponse, error) {
	// check required fields
	if err := fieldbehavior.ValidateRequiredFields(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	size, offset, perr := utils.ExtractPagination(in.PageSize, in.PageToken, s.Pagination)
	if perr != nil {
		return nil, perr
	}
	subsys, ok := s.Subsystems[in.Parent]
	if !ok {
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.Parent)
		return nil, err
	}
	// TODO: fix hard-coded Cntlid
	params := models.NvdaControllerNvmeNamespaceListParams{
		Subnqn: subsys.Spec.Nqn,
		Cntlid: 0,
	}
	var result models.NvdaControllerNvmeNamespaceListResult
	err := s.rpc.Call(ctx, "controller_nvme_namespace_list", &params, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	token, hasMoreElements := "", false
	log.Printf("Limiting result len(%d) to [%d:%d]", len(result.Namespaces), offset, size)
	result.Namespaces, hasMoreElements = utils.LimitPagination(result.Namespaces, offset, size)
	if hasMoreElements {
		token = uuid.New().String()
		s.Pagination[token] = offset + size
	}
	Blobarray := make([]*pb.NvmeNamespace, len(result.Namespaces))
	for i := range result.Namespaces {
		r := &result.Namespaces[i]
		Blobarray[i] = &pb.NvmeNamespace{Spec: &pb.NvmeNamespaceSpec{HostNsid: int32(r.Nsid)}}
	}
	sortNvmeNamespaces(Blobarray)
	return &pb.ListNvmeNamespacesResponse{NvmeNamespaces: Blobarray}, nil
}

// GetNvmeNamespace gets an Nvme namespace
func (s *Server) GetNvmeNamespace(ctx context.Context, in *pb.GetNvmeNamespaceRequest) (*pb.NvmeNamespace, error) {
	// check input correctness
	if err := s.validateGetNvmeNamespaceRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	namespace, ok := s.Namespaces[in.Name]
	if !ok {
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.Name)
		return nil, err
	}
	subsysName := utils.ResourceIDToSubsystemName(
		utils.GetSubsystemIDFromNvmeName(in.Name),
	)
	subsys, ok := s.Subsystems[subsysName]
	if !ok {
		err := status.Errorf(codes.NotFound, "unable to find key %s", subsysName)
		return nil, err
	}
	// TODO: fix hard-coded Cntlid
	params := models.NvdaControllerNvmeNamespaceListParams{
		Subnqn: subsys.Spec.Nqn,
		Cntlid: 0,
	}
	var result models.NvdaControllerNvmeNamespaceListResult
	err := s.rpc.Call(ctx, "controller_nvme_namespace_list", &params, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for i := range result.Namespaces {
		r := &result.Namespaces[i]
		if r.Nsid == int(namespace.Spec.HostNsid) {
			return &pb.NvmeNamespace{
				Spec: &pb.NvmeNamespaceSpec{
					HostNsid: int32(r.Nsid),
				},
				Status: &pb.NvmeNamespaceStatus{
					State:     pb.NvmeNamespaceStatus_STATE_ENABLED,
					OperState: pb.NvmeNamespaceStatus_OPER_STATE_ONLINE,
				},
			}, nil
		}
	}
	msg := fmt.Sprintf("Could not find HostNsid: %d", namespace.Spec.HostNsid)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

// StatsNvmeNamespace gets an Nvme namespace stats
func (s *Server) StatsNvmeNamespace(ctx context.Context, in *pb.StatsNvmeNamespaceRequest) (*pb.StatsNvmeNamespaceResponse, error) {
	// check input correctness
	if err := s.validateStatsNvmeNamespaceRequest(in); err != nil {
		return nil, err
	}
	// fetch object from the database
	namespace, ok := s.Namespaces[in.Name]
	if !ok {
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.Name)
		return nil, err
	}
	var result models.NvdaControllerNvmeStatsResult
	err := s.rpc.Call(ctx, "controller_nvme_get_iostat", nil, &result)
	if err != nil {
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for _, c := range result.Controllers {
		for _, r := range c.Bdevs {
			if r.BdevName == namespace.Spec.VolumeNameRef {
				return &pb.StatsNvmeNamespaceResponse{Stats: &pb.VolumeStats{
					ReadOpsCount:  int32(r.ReadIos),
					WriteOpsCount: int32(r.WriteIos),
				}}, nil
			}
		}
	}
	msg := fmt.Sprintf("Could not find BdevName: %s", namespace.Spec.VolumeNameRef)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}
