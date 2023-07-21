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

	"github.com/opiproject/gospdk/spdk"
	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/server"
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
	server.CloseListener(e.ln)
	server.CloseGrpcConnection(e.conn)
	if err := os.RemoveAll(e.testSocket); err != nil {
		log.Fatal(err)
	}
}

func createTestEnvironment(spdkResponses []string) *testEnv {
	env := &testEnv{}
	env.testSocket = server.GenerateSocketName("frontend")
	env.ln, env.jsonRPC = server.CreateTestSpdkServer(env.testSocket, spdkResponses)
	env.opiSpdkServer = NewServer(env.jsonRPC)

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
	testSubsystemName = server.ResourceIDToVolumeName(testSubsystemID)
	testSubsystem     = pb.NvmeSubsystem{
		Spec: &pb.NvmeSubsystemSpec{
			Nqn: "nqn.2022-09.io.spdk:opi3",
		},
	}
	testControllerID   = "controller-test"
	testControllerName = server.ResourceIDToVolumeName(testControllerID)
	testController     = pb.NvmeController{
		Spec: &pb.NvmeControllerSpec{
			SubsystemId:      &pc.ObjectKey{Value: testSubsystemName},
			PcieId:           &pb.PciEndpoint{PhysicalFunction: 1, VirtualFunction: 2},
			NvmeControllerId: 17,
		},
		Status: &pb.NvmeControllerStatus{
			Active: true,
		},
	}
	testNamespaceID   = "namespace-test"
	testNamespaceName = server.ResourceIDToVolumeName(testNamespaceID)
	testNamespace     = pb.NvmeNamespace{
		Spec: &pb.NvmeNamespaceSpec{
			HostNsid:    22,
			SubsystemId: &pc.ObjectKey{Value: testSubsystemName},
			VolumeId:    &pc.ObjectKey{Value: "Malloc1"},
		},
		Status: &pb.NvmeNamespaceStatus{
			PciState:     2,
			PciOperState: 1,
		},
	}
	testVirtioCtrlID   = "virtio-blk-42"
	testVirtioCtrlName = server.ResourceIDToVolumeName(testVirtioCtrlID)
	testVirtioCtrl     = pb.VirtioBlk{
		PcieId:   &pb.PciEndpoint{PhysicalFunction: 42},
		VolumeId: &pc.ObjectKey{Value: "Malloc42"},
		MaxIoQps: 1,
	}
)
