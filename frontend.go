// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.

package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/ulule/deepcopier"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	pb.UnimplementedFrontendNvmeServiceServer
}

var PluginFrontendNvme server

// ////////////////////////////////////////////////////////
var subsystems = map[string]*pb.NVMeSubsystem{}

func (s *server) CreateNVMeSubsystem(ctx context.Context, in *pb.CreateNVMeSubsystemRequest) (*pb.NVMeSubsystem, error) {
	log.Printf("CreateNVMeSubsystem: Received from client: %v", in)
	params := NvdaSubsystemNvmeCreateParams{
		Nqn:          in.NvMeSubsystem.Spec.Nqn,
		SerialNumber: in.NvMeSubsystem.Spec.SerialNumber,
		ModelNumber:  in.NvMeSubsystem.Spec.ModelNumber,
	}
	var result NvdaSubsystemNvmeCreateResult
	err := call("subsystem_nvme_create", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	subsystems[in.NvMeSubsystem.Spec.Id.Value] = in.NvMeSubsystem
	log.Printf("Received from SPDK: %v", result)
	response := &pb.NVMeSubsystem{}
	err = deepcopier.Copy(in.NvMeSubsystem).To(response)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	return response, nil
}

func (s *server) DeleteNVMeSubsystem(ctx context.Context, in *pb.DeleteNVMeSubsystemRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteNVMeSubsystem: Received from client: %v", in)
	subsys, ok := subsystems[in.Name]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Name)
		log.Printf("error: %v", err)
		return nil, err
	}
	params := NvdaSubsystemNvmeDeleteParams{
		Nqn: subsys.Spec.Nqn,
	}
	var result NvdaSubsystemNvmeDeleteResult
	err := call("subsystem_nvme_delete", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	delete(subsystems, subsys.Spec.Id.Value)
	return &emptypb.Empty{}, nil
}

func (s *server) UpdateNVMeSubsystem(ctx context.Context, in *pb.UpdateNVMeSubsystemRequest) (*pb.NVMeSubsystem, error) {
	log.Printf("UpdateNVMeSubsystem: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNVMeSubsystem method is not implemented")
}

func (s *server) ListNVMeSubsystems(ctx context.Context, in *pb.ListNVMeSubsystemsRequest) (*pb.ListNVMeSubsystemsResponse, error) {
	log.Printf("ListNVMeSubsystems: Received from client: %v", in)
	var result []NvdaSubsystemNvmeListResult
	err := call("subsystem_nvme_list", nil, &result)
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

func (s *server) GetNVMeSubsystem(ctx context.Context, in *pb.GetNVMeSubsystemRequest) (*pb.NVMeSubsystem, error) {
	log.Printf("GetNVMeSubsystem: Received from client: %v", in)
	subsys, ok := subsystems[in.Name]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Name)
		log.Printf("error: %v", err)
		return nil, err
	}
	var result []NvdaSubsystemNvmeListResult
	err := call("subsystem_nvme_list", nil, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for i := range result {
		r := &result[i]
		if r.Nqn == subsys.Spec.Nqn {
			return &pb.NVMeSubsystem{Spec: &pb.NVMeSubsystemSpec{Nqn: r.Nqn, SerialNumber: r.SerialNumber, ModelNumber: r.ModelNumber}}, nil
		}
	}
	msg := fmt.Sprintf("Could not find NQN: %s", subsys.Spec.Nqn)
	log.Print(msg)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

func (s *server) NVMeSubsystemStats(ctx context.Context, in *pb.NVMeSubsystemStatsRequest) (*pb.NVMeSubsystemStatsResponse, error) {
	log.Printf("NVMeSubsystemStats: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNVMeSubsystem method is not implemented")
}

// ////////////////////////////////////////////////////////
var controllers = map[string]*pb.NVMeController{}

func (s *server) CreateNVMeController(ctx context.Context, in *pb.CreateNVMeControllerRequest) (*pb.NVMeController, error) {
	log.Printf("CreateNVMeController: Received from client: %v", in)
	subsys, ok := subsystems[in.NvMeController.Spec.SubsystemId.Value]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.NvMeController.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}

	params := NvdaControllerNvmeCreateParams{
		Nqn:              subsys.Spec.Nqn,
		EmulationManager: "mlx5_0",
		PfID:             int(in.NvMeController.Spec.PcieId.PhysicalFunction),
		// VfID:             int(in.NvMeController.Spec.PcieId.VirtualFunction),
		// MaxNamespaces:    int(in.NvMeController.Spec.MaxNsq),
		// NrIoQueues:       int(in.NvMeController.Spec.MaxNcq),
	}
	var result NvdaControllerNvmeCreateResult
	err := call("controller_nvme_create", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	controllers[in.NvMeController.Spec.Id.Value] = in.NvMeController
	controllers[in.NvMeController.Spec.Id.Value].Spec.NvmeControllerId = int32(result.Cntlid)
	response := &pb.NVMeController{Spec: &pb.NVMeControllerSpec{Id: &pc.ObjectKey{Value: "TBD"}}}
	err = deepcopier.Copy(in.NvMeController).To(response)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	return response, nil
}

func (s *server) DeleteNVMeController(ctx context.Context, in *pb.DeleteNVMeControllerRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteNVMeController: Received from client: %v", in)
	controller, ok := controllers[in.Name]
	if !ok {
		return nil, fmt.Errorf("error finding controller %s", in.Name)
	}
	subsys, ok := subsystems[controller.Spec.SubsystemId.Value]
	if !ok {
		err := fmt.Errorf("unable to find key %s", controller.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}

	params := NvdaControllerNvmeDeleteParams{
		Subnqn: subsys.Spec.Nqn,
		Cntlid: int(controller.Spec.NvmeControllerId),
	}
	var result NvdaControllerNvmeDeleteResult
	err := call("controller_nvme_delete", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	delete(controllers, controller.Spec.Id.Value)
	return &emptypb.Empty{}, nil
}

func (s *server) UpdateNVMeController(ctx context.Context, in *pb.UpdateNVMeControllerRequest) (*pb.NVMeController, error) {
	log.Printf("UpdateNVMeController: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNVMeController method is not implemented")
}

func (s *server) ListNVMeControllers(ctx context.Context, in *pb.ListNVMeControllersRequest) (*pb.ListNVMeControllersResponse, error) {
	log.Printf("ListNVMeControllers: Received from client: %v", in)
	subsys, ok := subsystems[in.Parent]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Parent)
		log.Printf("error: %v", err)
		return nil, err
	}
	var result []NvdaControllerNvmeListResult
	err := call("controller_list", nil, &result)
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

func (s *server) GetNVMeController(ctx context.Context, in *pb.GetNVMeControllerRequest) (*pb.NVMeController, error) {
	log.Printf("GetNVMeController: Received from client: %v", in)
	controller, ok := controllers[in.Name]
	if !ok {
		return nil, fmt.Errorf("error finding controller %s", in.Name)
	}
	var result []NvdaControllerNvmeListResult
	err := call("controller_list", nil, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for i := range result {
		r := &result[i]
		if r.Cntlid == int(controller.Spec.NvmeControllerId) && r.Type == "nvme" {
			return &pb.NVMeController{Spec: &pb.NVMeControllerSpec{NvmeControllerId: int32(r.Cntlid)}}, nil
		}
	}
	msg := fmt.Sprintf("Could not find NvmeControllerId: %d", controller.Spec.NvmeControllerId)
	log.Print(msg)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

func (s *server) NVMeControllerStats(ctx context.Context, in *pb.NVMeControllerStatsRequest) (*pb.NVMeControllerStatsResponse, error) {
	log.Printf("NVMeControllerStats: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNVMeSubsystem method is not implemented")
}

// ////////////////////////////////////////////////////////
var namespaces = map[string]*pb.NVMeNamespace{}

func (s *server) CreateNVMeNamespace(ctx context.Context, in *pb.CreateNVMeNamespaceRequest) (*pb.NVMeNamespace, error) {
	log.Printf("CreateNVMeNamespace: Received from client: %v", in)
	subsys, ok := subsystems[in.NvMeNamespace.Spec.SubsystemId.Value]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.NvMeNamespace.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}
	// TODO: do lookup through VolumeId key instead of using it's value
	params := NvdaControllerNvmeNamespaceAttachParams{
		BdevType: "spdk",
		Bdev:     in.NvMeNamespace.Spec.VolumeId.Value,
		Nsid:     int(in.NvMeNamespace.Spec.HostNsid),
		Subnqn:   subsys.Spec.Nqn,
		Cntlid:   0,
		UUID:     in.NvMeNamespace.Spec.Uuid.Value,
		Nguid:    in.NvMeNamespace.Spec.Nguid,
		Eui64:    strconv.FormatInt(in.NvMeNamespace.Spec.Eui64, 10),
	}
	var result NvdaControllerNvmeNamespaceAttachResult
	err := call("controller_nvme_namespace_attach", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	namespaces[in.NvMeNamespace.Spec.Id.Value] = in.NvMeNamespace

	response := &pb.NVMeNamespace{}
	err = deepcopier.Copy(in.NvMeNamespace).To(response)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	return response, nil
}

func (s *server) DeleteNVMeNamespace(ctx context.Context, in *pb.DeleteNVMeNamespaceRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteNVMeNamespace: Received from client: %v", in)
	namespace, ok := namespaces[in.Name]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Name)
		log.Printf("error: %v", err)
		return nil, err
	}
	subsys, ok := subsystems[namespace.Spec.SubsystemId.Value]
	if !ok {
		err := fmt.Errorf("unable to find subsystem %s", namespace.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}

	// TODO: fix hard-coded Cntlid
	params := NvdaControllerNvmeNamespaceDetachParams{
		Nsid:   int(namespace.Spec.HostNsid),
		Subnqn: subsys.Spec.Nqn,
		Cntlid: 0,
	}
	var result NvdaControllerNvmeNamespaceDetachResult
	err := call("controller_nvme_namespace_detach", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	delete(namespaces, namespace.Spec.Id.Value)
	return &emptypb.Empty{}, nil
}

func (s *server) UpdateNVMeNamespace(ctx context.Context, in *pb.UpdateNVMeNamespaceRequest) (*pb.NVMeNamespace, error) {
	log.Printf("UpdateNVMeNamespace: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "UpdateNVMeNamespace method is not implemented")
}

func (s *server) ListNVMeNamespaces(ctx context.Context, in *pb.ListNVMeNamespacesRequest) (*pb.ListNVMeNamespacesResponse, error) {
	log.Printf("ListNVMeNamespaces: Received from client: %v", in)
	subsys, ok := subsystems[in.Parent]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Parent)
		log.Printf("error: %v", err)
		return nil, err
	}
	// TODO: fix hard-coded Cntlid
	params := NvdaControllerNvmeNamespaceListParams{
		Subnqn: subsys.Spec.Nqn,
		Cntlid: 0,
	}
	var result NvdaControllerNvmeNamespaceListResult
	err := call("controller_nvme_namespace_list", &params, &result)
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

func (s *server) GetNVMeNamespace(ctx context.Context, in *pb.GetNVMeNamespaceRequest) (*pb.NVMeNamespace, error) {
	log.Printf("GetNVMeNamespace: Received from client: %v", in)
	namespace, ok := namespaces[in.Name]
	if !ok {
		err := fmt.Errorf("unable to find key %s", in.Name)
		log.Printf("error: %v", err)
		return nil, err
	}
	subsys, ok := subsystems[namespace.Spec.SubsystemId.Value]
	if !ok {
		err := fmt.Errorf("unable to find key %s", namespace.Spec.SubsystemId.Value)
		log.Printf("error: %v", err)
		return nil, err
	}
	// TODO: fix hard-coded Cntlid
	params := NvdaControllerNvmeNamespaceListParams{
		Subnqn: subsys.Spec.Nqn,
		Cntlid: 0,
	}
	var result NvdaControllerNvmeNamespaceListResult
	err := call("controller_nvme_namespace_list", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, err
	}
	log.Printf("Received from SPDK: %v", result)
	for i := range result.Namespaces {
		r := &result.Namespaces[i]
		if r.Nsid == int(namespace.Spec.HostNsid) {
			return &pb.NVMeNamespace{Spec: &pb.NVMeNamespaceSpec{HostNsid: int32(r.Nsid)}}, nil
		}
	}
	msg := fmt.Sprintf("Could not find HostNsid: %d", namespace.Spec.HostNsid)
	log.Print(msg)
	return nil, status.Errorf(codes.InvalidArgument, msg)
}

func (s *server) NVMeNamespaceStats(ctx context.Context, in *pb.NVMeNamespaceStatsRequest) (*pb.NVMeNamespaceStatsResponse, error) {
	log.Printf("NVMeNamespaceStats: Received from client: %v", in)
	return nil, status.Errorf(codes.Unimplemented, "NVMeNamespaceStats method is not implemented")
}
