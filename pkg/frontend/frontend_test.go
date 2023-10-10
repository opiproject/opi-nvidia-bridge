// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"context"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/philippgille/gokv/gomap"

	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/utils"
)

type frontendClient struct {
	pb.FrontendNvmeServiceClient
	pb.FrontendVirtioBlkServiceClient
}

type testEnv struct {
	opiSpdkServer *Server
	client        *frontendClient
	ln            net.Listener
	testSocket    string
	ctx           context.Context
	conn          *grpc.ClientConn
	jsonRPC       spdk.JSONRPC
}

func (e *testEnv) Close() {
	utils.CloseListener(e.ln)
	utils.CloseGrpcConnection(e.conn)
	if err := os.RemoveAll(e.testSocket); err != nil {
		log.Fatal(err)
	}
}

func createTestEnvironment(spdkResponses []string) *testEnv {
	env := &testEnv{}
	env.testSocket = utils.GenerateSocketName("frontend")
	env.ln, env.jsonRPC = utils.CreateTestSpdkServer(env.testSocket, spdkResponses)
	options := gomap.DefaultOptions
	options.Codec = utils.ProtoCodec{}
	store := gomap.NewStore(options)
	env.opiSpdkServer = NewServer(env.jsonRPC, store)

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx,
		"",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer(env.opiSpdkServer)))
	if err != nil {
		log.Fatal(err)
	}
	env.ctx = ctx
	env.conn = conn

	env.client = &frontendClient{
		pb.NewFrontendNvmeServiceClient(env.conn),
		pb.NewFrontendVirtioBlkServiceClient(env.conn),
	}

	return env
}

func dialer(opiSpdkServer *Server) func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	pb.RegisterFrontendNvmeServiceServer(server, opiSpdkServer)
	pb.RegisterFrontendVirtioBlkServiceServer(server, opiSpdkServer)

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

var (
	testSubsystemID   = "subsystem-test"
	testSubsystemName = utils.ResourceIDToSubsystemName(testSubsystemID)
	testSubsystem     = pb.NvmeSubsystem{
		Spec: &pb.NvmeSubsystemSpec{
			Nqn: "nqn.2022-09.io.spdk:opi3",
		},
	}
	testSubsystemWithStatus = pb.NvmeSubsystem{
		Name: testSubsystemName,
		Spec: testSubsystem.Spec,
		Status: &pb.NvmeSubsystemStatus{
			FirmwareRevision: "TBD",
		},
	}

	testControllerID   = "controller-test"
	testControllerName = utils.ResourceIDToControllerName(testSubsystemID, testControllerID)
	testController     = pb.NvmeController{
		Spec: &pb.NvmeControllerSpec{
			Endpoint: &pb.NvmeControllerSpec_PcieId{
				PcieId: &pb.PciEndpoint{
					PhysicalFunction: wrapperspb.Int32(1),
					VirtualFunction:  wrapperspb.Int32(2),
					PortId:           wrapperspb.Int32(0),
				},
			},
			Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_PCIE,
			NvmeControllerId: proto.Int32(17),
		},
	}
	testControllerWithStatus = pb.NvmeController{
		Name: testControllerName,
		Spec: testController.Spec,
		Status: &pb.NvmeControllerStatus{
			Active: true,
		},
	}

	testNamespaceID   = "namespace-test"
	testNamespaceName = utils.ResourceIDToNamespaceName(testSubsystemID, testNamespaceID)
	testNamespace     = pb.NvmeNamespace{
		Spec: &pb.NvmeNamespaceSpec{
			HostNsid:      22,
			VolumeNameRef: "Malloc1",
		},
	}
	testNamespaceWithStatus = pb.NvmeNamespace{
		Name: testNamespaceName,
		Spec: testNamespace.Spec,
		Status: &pb.NvmeNamespaceStatus{
			State:     pb.NvmeNamespaceStatus_STATE_ENABLED,
			OperState: pb.NvmeNamespaceStatus_OPER_STATE_ONLINE,
		},
	}
	testVirtioCtrlID   = "virtio-blk-42"
	testVirtioCtrlName = utils.ResourceIDToVolumeName(testVirtioCtrlID)
	testVirtioCtrl     = pb.VirtioBlk{
		PcieId: &pb.PciEndpoint{
			PhysicalFunction: wrapperspb.Int32(42),
			VirtualFunction:  wrapperspb.Int32(0),
			PortId:           wrapperspb.Int32(0),
		},
		VolumeNameRef: "Malloc42",
		MaxIoQps:      1,
	}

	checkGlobalTestProtoObjectsNotChanged = utils.CheckTestProtoObjectsNotChanged(
		&testSubsystem,
		&testController,
		&testNamespace,
		&testVirtioCtrl,
		&testSubsystemWithStatus,
		&testControllerWithStatus,
		&testNamespaceWithStatus,
	)
)
