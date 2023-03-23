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
	"strconv"

	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-nvidia-bridge/pkg/models"
	spdk "github.com/opiproject/opi-spdk-bridge/pkg/models"

	"github.com/ulule/deepcopier"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CreateNVMeSubsystem creates an NVMe Subsystem
func (s *Server) CreateNVMeSubsystem(_ context.Context, in *pb.CreateNVMeSubsystemRequest) (*pb.NVMeSubsystem, error) {
	log.Printf("CreateNVMeSubsystem: Received from client: %v", in)
	// idempotent API when called with same key, should return same object
	subsys, ok := s.Subsystems[in.NvMeSubsystem.Spec.Id.Value]
	if ok {
		log.Printf("Already existing NVMeSubsystem with id %v", in.NvMeSubsystem.Spec.Id.Value)
		return subsys, nil
	}
	// not found, so create a new one
	params := models.NvdaSubsystemNvmeCreateParams{
		Nqn:          in.NvMeSubsystem.Spec.Nqn,
		SerialNumber: in.NvMeSubsystem.Spec.SerialNumber,
		ModelNumber:  in.NvMeSubsystem.Spec.ModelNumber,
	}
	var result models.NvdaSubsystemNvmeCreateResult
	err := s.rpc.Call("subsystem_nvme_create", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	s.Subsystems[in.NvMeSubsystem.Spec.Id.Value] = in.NvMeSubsystem
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not create NQN: %s", in.NvMeSubsystem.Spec.Nqn)
		log.Print(msg)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	var ver spdk.GetVersionResult
	err = s.rpc.Call("spdk_get_version", nil, &ver)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", ver)
	response := &pb.NVMeSubsystem{}
	err = deepcopier.Copy(in.NvMeSubsystem).To(response)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	response.Status = &pb.NVMeSubsystemStatus{FirmwareRevision: ver.Version}
	return response, nil
}

// DeleteNVMeSubsystem deletes an NVMe Subsystem
func (s *Server) DeleteNVMeSubsystem(_ context.Context, in *pb.DeleteNVMeSubsystemRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteNVMeSubsystem: Received from client: %v", in)
	subsys, ok := s.Subsystems[in.Name]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Name)
		log.Printf("error: %v", err)
		return nil, err
	}
	params := models.NvdaSubsystemNvmeDeleteParams{
		Nqn: subsys.Spec.Nqn,
	}
	var result models.NvdaSubsystemNvmeDeleteResult
	err := s.rpc.Call("subsystem_nvme_delete", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not delete NQN: %s", subsys.Spec.Nqn)
		log.Print(msg)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	delete(s.Subsystems, subsys.Spec.Id.Value)
	return &emptypb.Empty{}, nil
}

// UpdateNVMeSubsystem updates an NVMe Subsystem
func (s *Server) UpdateNVMeSubsystem(_ context.Context, in *pb.UpdateNVMeSubsystemRequest) (*pb.NVMeSubsystem, error) {
	log.Printf("UpdateNVMeSubsystem: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNVMeSubsystem method is not implemented")
}

// ListNVMeSubsystems lists NVMe Subsystems
func (s *Server) ListNVMeSubsystems(_ context.Context, in *pb.ListNVMeSubsystemsRequest) (*pb.ListNVMeSubsystemsResponse, error) {
	log.Printf("ListNVMeSubsystems: Received from client: %v", in)
	var result []models.NvdaSubsystemNvmeListResult
	err := s.rpc.Call("subsystem_nvme_list", nil, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	Blobarray := make([]*pb.NVMeSubsystem, len(result))
	for i := range result {
		r := &result[i]
		Blobarray[i] = &pb.NVMeSubsystem{Spec: &pb.NVMeSubsystemSpec{Nqn: r.Nqn, SerialNumber: r.SerialNumber, ModelNumber: r.ModelNumber}}
	}
	return &pb.ListNVMeSubsystemsResponse{NvMeSubsystems: Blobarray}, nil
}

// GetNVMeSubsystem gets NVMe Subsystems
func (s *Server) GetNVMeSubsystem(_ context.Context, in *pb.GetNVMeSubsystemRequest) (*pb.NVMeSubsystem, error) {
	log.Printf("GetNVMeSubsystem: Received from client: %v", in)
	subsys, ok := s.Subsystems[in.Name]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Name)
		log.Printf("error: %v", err)
		return nil, err
	}
	var result []models.NvdaSubsystemNvmeListResult
	err := s.rpc.Call("subsystem_nvme_list", nil, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for i := range result {
		r := &result[i]
		if r.Nqn == subsys.Spec.Nqn {
			return &pb.NVMeSubsystem{Spec: &pb.NVMeSubsystemSpec{Nqn: r.Nqn, SerialNumber: r.SerialNumber, ModelNumber: r.ModelNumber}, Status: &pb.NVMeSubsystemStatus{FirmwareRevision: "TBD"}}, nil
		}
	}
	msg := fmt.Sprintf("Could not find NQN: %s", subsys.Spec.Nqn)
	log.Print(msg)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

// NVMeSubsystemStats gets NVMe Subsystem stats
func (s *Server) NVMeSubsystemStats(_ context.Context, in *pb.NVMeSubsystemStatsRequest) (*pb.NVMeSubsystemStatsResponse, error) {
	log.Printf("NVMeSubsystemStats: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNVMeSubsystem method is not implemented")
}

// CreateNVMeController creates an NVMe controller
func (s *Server) CreateNVMeController(_ context.Context, in *pb.CreateNVMeControllerRequest) (*pb.NVMeController, error) {
	log.Printf("CreateNVMeController: Received from client: %v", in)
	// idempotent API when called with same key, should return same object
	controller, ok := s.Controllers[in.NvMeController.Spec.Id.Value]
	if ok {
		log.Printf("Already existing NVMeController with id %v", in.NvMeController.Spec.Id.Value)
		return controller, nil
	}
	// not found, so create a new one
	subsys, ok := s.Subsystems[in.NvMeController.Spec.SubsystemId.Value]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.NvMeController.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}

	params := models.NvdaControllerNvmeCreateParams{
		Nqn:              subsys.Spec.Nqn,
		EmulationManager: "mlx5_0",
		PfID:             int(in.NvMeController.Spec.PcieId.PhysicalFunction),
		// VfID:             int(in.NvMeController.Spec.PcieId.VirtualFunction),
		// MaxNamespaces:    int(in.NvMeController.Spec.MaxNsq),
		// NrIoQueues:       int(in.NvMeController.Spec.MaxNcq),
	}
	var result models.NvdaControllerNvmeCreateResult
	err := s.rpc.Call("controller_nvme_create", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if result.Cntlid < 0 {
		msg := fmt.Sprintf("Could not create CTRL: %s", in.NvMeController.Spec.Id.Value)
		log.Print(msg)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	s.Controllers[in.NvMeController.Spec.Id.Value] = in.NvMeController
	s.Controllers[in.NvMeController.Spec.Id.Value].Spec.NvmeControllerId = int32(result.Cntlid)
	s.Controllers[in.NvMeController.Spec.Id.Value].Status = &pb.NVMeControllerStatus{Active: true}
	response := &pb.NVMeController{Spec: &pb.NVMeControllerSpec{Id: &pc.ObjectKey{Value: "TBD"}}}
	err = deepcopier.Copy(in.NvMeController).To(response)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	return response, nil
}

// DeleteNVMeController deletes an NVMe controller
func (s *Server) DeleteNVMeController(_ context.Context, in *pb.DeleteNVMeControllerRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteNVMeController: Received from client: %v", in)
	controller, ok := s.Controllers[in.Name]
	if !ok {
		return nil, fmt.Errorf("error finding controller %s", in.Name)
	}
	subsys, ok := s.Subsystems[controller.Spec.SubsystemId.Value]
	if !ok {
		err := fmt.Errorf("unable to find key %s", controller.Spec.SubsystemId.Value)
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
	delete(s.Controllers, controller.Spec.Id.Value)
	return &emptypb.Empty{}, nil
}

// UpdateNVMeController updates an NVMe controller
func (s *Server) UpdateNVMeController(_ context.Context, in *pb.UpdateNVMeControllerRequest) (*pb.NVMeController, error) {
	log.Printf("UpdateNVMeController: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNVMeController method is not implemented")
}

// ListNVMeControllers lists NVMe controllers
func (s *Server) ListNVMeControllers(_ context.Context, in *pb.ListNVMeControllersRequest) (*pb.ListNVMeControllersResponse, error) {
	log.Printf("ListNVMeControllers: Received from client: %v", in)
	subsys, ok := s.Subsystems[in.Parent]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Parent)
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
	Blobarray := make([]*pb.NVMeController, len(result))
	for i := range result {
		r := &result[i]
		if r.Subnqn == subsys.Spec.Nqn && r.Type == "nvme" {
			Blobarray[i] = &pb.NVMeController{Spec: &pb.NVMeControllerSpec{NvmeControllerId: int32(r.Cntlid)}}
		}
	}
	return &pb.ListNVMeControllersResponse{NvMeControllers: Blobarray}, nil
}

// GetNVMeController gets an NVMe controller
func (s *Server) GetNVMeController(_ context.Context, in *pb.GetNVMeControllerRequest) (*pb.NVMeController, error) {
	log.Printf("GetNVMeController: Received from client: %v", in)
	controller, ok := s.Controllers[in.Name]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Name)
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
			return &pb.NVMeController{Spec: &pb.NVMeControllerSpec{NvmeControllerId: int32(r.Cntlid)}, Status: &pb.NVMeControllerStatus{Active: true}}, nil
		}
	}
	msg := fmt.Sprintf("Could not find NvmeControllerId: %d", controller.Spec.NvmeControllerId)
	log.Print(msg)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

// NVMeControllerStats gets an NVMe controller stats
func (s *Server) NVMeControllerStats(_ context.Context, in *pb.NVMeControllerStatsRequest) (*pb.NVMeControllerStatsResponse, error) {
	log.Printf("NVMeControllerStats: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "NVMeControllerStats method is not implemented")
}

// CreateNVMeNamespace creates an NVMe namespace
func (s *Server) CreateNVMeNamespace(_ context.Context, in *pb.CreateNVMeNamespaceRequest) (*pb.NVMeNamespace, error) {
	log.Printf("CreateNVMeNamespace: Received from client: %v", in)
	// idempotent API when called with same key, should return same object
	namespace, ok := s.Namespaces[in.NvMeNamespace.Spec.Id.Value]
	if ok {
		log.Printf("Already existing NVMeNamespace with id %v", in.NvMeNamespace.Spec.Id.Value)
		return namespace, nil
	}
	// not found, so create a new one
	subsys, ok := s.Subsystems[in.NvMeNamespace.Spec.SubsystemId.Value]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.NvMeNamespace.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}
	// TODO: do lookup through VolumeId key instead of using it's value
	params := models.NvdaControllerNvmeNamespaceAttachParams{
		BdevType: "spdk",
		Bdev:     in.NvMeNamespace.Spec.VolumeId.Value,
		Nsid:     int(in.NvMeNamespace.Spec.HostNsid),
		Subnqn:   subsys.Spec.Nqn,
		Cntlid:   0,
		UUID:     in.NvMeNamespace.Spec.Uuid.Value,
		Nguid:    in.NvMeNamespace.Spec.Nguid,
		Eui64:    strconv.FormatInt(in.NvMeNamespace.Spec.Eui64, 10),
	}
	var result models.NvdaControllerNvmeNamespaceAttachResult
	err := s.rpc.Call("controller_nvme_namespace_attach", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not create NS: %s", in.NvMeNamespace.Spec.Id.Value)
		log.Print(msg)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	s.Namespaces[in.NvMeNamespace.Spec.Id.Value] = in.NvMeNamespace
	response := &pb.NVMeNamespace{}
	err = deepcopier.Copy(in.NvMeNamespace).To(response)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	response.Status = &pb.NVMeNamespaceStatus{PciState: 2, PciOperState: 1}
	return response, nil
}

// DeleteNVMeNamespace deletes an NVMe namespace
func (s *Server) DeleteNVMeNamespace(_ context.Context, in *pb.DeleteNVMeNamespaceRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteNVMeNamespace: Received from client: %v", in)
	namespace, ok := s.Namespaces[in.Name]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Name)
		log.Printf("error: %v", err)
		return nil, err
	}
	subsys, ok := s.Subsystems[namespace.Spec.SubsystemId.Value]
	if !ok {
		err := fmt.Errorf("unable to find subsystem %s", namespace.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}

	// TODO: fix hard-coded Cntlid
	params := models.NvdaControllerNvmeNamespaceDetachParams{
		Nsid:   int(namespace.Spec.HostNsid),
		Subnqn: subsys.Spec.Nqn,
		Cntlid: 0,
	}
	var result models.NvdaControllerNvmeNamespaceDetachResult
	err := s.rpc.Call("controller_nvme_namespace_detach", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not delete NS: %s", namespace.Spec.SubsystemId.Value)
		log.Print(msg)
		return nil, status.Errorf(codes.InvalidArgument, msg)
	}
	delete(s.Namespaces, namespace.Spec.Id.Value)
	return &emptypb.Empty{}, nil
}

// UpdateNVMeNamespace updates an NVMe namespace
func (s *Server) UpdateNVMeNamespace(_ context.Context, in *pb.UpdateNVMeNamespaceRequest) (*pb.NVMeNamespace, error) {
	log.Printf("UpdateNVMeNamespace: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNVMeNamespace method is not implemented")
}

// ListNVMeNamespaces lists NVMe namespaces
func (s *Server) ListNVMeNamespaces(_ context.Context, in *pb.ListNVMeNamespacesRequest) (*pb.ListNVMeNamespacesResponse, error) {
	log.Printf("ListNVMeNamespaces: Received from client: %v", in)
	subsys, ok := s.Subsystems[in.Parent]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Parent)
		log.Printf("error: %v", err)
		return nil, err
	}
	// TODO: fix hard-coded Cntlid
	params := models.NvdaControllerNvmeNamespaceListParams{
		Subnqn: subsys.Spec.Nqn,
		Cntlid: 0,
	}
	var result models.NvdaControllerNvmeNamespaceListResult
	err := s.rpc.Call("controller_nvme_namespace_list", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	Blobarray := make([]*pb.NVMeNamespace, len(result.Namespaces))
	for i := range result.Namespaces {
		r := &result.Namespaces[i]
		Blobarray[i] = &pb.NVMeNamespace{Spec: &pb.NVMeNamespaceSpec{HostNsid: int32(r.Nsid)}}
	}
	return &pb.ListNVMeNamespacesResponse{NvMeNamespaces: Blobarray}, nil
}

// GetNVMeNamespace gets an NVMe namespace
func (s *Server) GetNVMeNamespace(_ context.Context, in *pb.GetNVMeNamespaceRequest) (*pb.NVMeNamespace, error) {
	log.Printf("GetNVMeNamespace: Received from client: %v", in)
	namespace, ok := s.Namespaces[in.Name]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Name)
		log.Printf("error: %v", err)
		return nil, err
	}
	subsys, ok := s.Subsystems[namespace.Spec.SubsystemId.Value]
	if !ok {
		err := fmt.Errorf("unable to find key %s", namespace.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}
	// TODO: fix hard-coded Cntlid
	params := models.NvdaControllerNvmeNamespaceListParams{
		Subnqn: subsys.Spec.Nqn,
		Cntlid: 0,
	}
	var result models.NvdaControllerNvmeNamespaceListResult
	err := s.rpc.Call("controller_nvme_namespace_list", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for i := range result.Namespaces {
		r := &result.Namespaces[i]
		if r.Nsid == int(namespace.Spec.HostNsid) {
			return &pb.NVMeNamespace{Spec: &pb.NVMeNamespaceSpec{HostNsid: int32(r.Nsid)}, Status: &pb.NVMeNamespaceStatus{PciState: 2, PciOperState: 1}}, nil
		}
	}
	msg := fmt.Sprintf("Could not find HostNsid: %d", namespace.Spec.HostNsid)
	log.Print(msg)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

// NVMeNamespaceStats gets an NVMe namespace stats
func (s *Server) NVMeNamespaceStats(_ context.Context, in *pb.NVMeNamespaceStatsRequest) (*pb.NVMeNamespaceStatsResponse, error) {
	log.Printf("NVMeNamespaceStats: Received from client: %v", in)
	namespace, ok := s.Namespaces[in.NamespaceId.Value]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.NamespaceId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}
	var result models.NvdaControllerNvmeStatsResult
	err := s.rpc.Call("controller_nvme_get_iostat", nil, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for _, c := range result.Controllers {
		for _, r := range c.Bdevs {
			if r.BdevName == namespace.Spec.VolumeId.Value {
				return &pb.NVMeNamespaceStatsResponse{Id: in.NamespaceId, Stats: &pb.VolumeStats{
					ReadOpsCount:  int32(r.ReadIos),
					WriteOpsCount: int32(r.WriteIos),
				}}, nil
			}
		}
	}
	msg := fmt.Sprintf("Could not find BdevName: %s", namespace.Spec.VolumeId.Value)
	log.Print(msg)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}
