// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"reflect"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/grpc/credentials/insecure"

	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
)

func dialer() func(context.Context, string) (net.Conn, error) {
	listener := bufconn.Listen(1024 * 1024)

	server := grpc.NewServer()

	pb.RegisterFrontendNvmeServiceServer(server, &PluginFrontendNvme)

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func spdkMockServer(l net.Listener, spdk string) {
	fd, err := l.Accept()
	if err != nil {
		log.Fatal("accept error:", err)
	}
	log.Printf("SPDK mockup server: client connected [%s]", fd.RemoteAddr().Network())

	buf := make([]byte, 512)
	nr, err := fd.Read(buf)
	if err != nil {
		return
	}

	data := buf[0:nr]

	log.Printf("SPDK mockup server: got : %s", string(data))
	log.Printf("SPDK mockup server: snd : %s", string(spdk))

	_, err = fd.Write([]byte(string(spdk)))
	if err != nil {
		log.Fatal("Write: ", err)
	}
	err = fd.(*net.UnixConn).CloseWrite()
	if err != nil {
		log.Fatal(err)
	}
}

func TestFrontEnd_CreateNVMeSubsystem(t *testing.T) {
	tests := []struct {
		name    string
		in      *pb.NVMeSubsystem
		out     *pb.NVMeSubsystem
		spdk    string
		errCode codes.Code
		errMsg  string
	}{
		{
			"valid request with invalid SPDK responce",
			&pb.NVMeSubsystem{
				Spec: &pb.NVMeSubsystemSpec{
					Id:           &pc.ObjectKey{Value: "subsystem-test"},
					Nqn:          "nqn.2022-09.io.spdk:opi3",
					SerialNumber: "OpiSerialNumber",
					ModelNumber:  "OpiModelNumber",
				},
			},
			nil,
			`{"id":1,"error":{"code":0,"message":""},"result":false}`,
			codes.InvalidArgument,
			fmt.Sprintf("Could not create NQN: %v", "nqn.2022-09.io.spdk:opi3"),
		},
		{
			"valid request with empty SPDK responce",
			&pb.NVMeSubsystem{
				Spec: &pb.NVMeSubsystemSpec{
					Id:           &pc.ObjectKey{Value: "subsystem-test"},
					Nqn:          "nqn.2022-09.io.spdk:opi3",
					SerialNumber: "OpiSerialNumber",
					ModelNumber:  "OpiModelNumber",
				},
			},
			nil,
			"",
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_create: %v", "EOF"),
		},
		{
			"valid request with ID mismatch SPDK responce",
			&pb.NVMeSubsystem{
				Spec: &pb.NVMeSubsystemSpec{
					Id:           &pc.ObjectKey{Value: "subsystem-test"},
					Nqn:          "nqn.2022-09.io.spdk:opi3",
					SerialNumber: "OpiSerialNumber",
					ModelNumber:  "OpiModelNumber",
				},
			},
			nil,
			`{"id":0,"error":{"code":0,"message":""},"result":false}`,
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_create: %v", "json response ID mismatch"),
		},
		{
			"valid request with error code from SPDK responce",
			&pb.NVMeSubsystem{
				Spec: &pb.NVMeSubsystemSpec{
					Id:           &pc.ObjectKey{Value: "subsystem-test"},
					Nqn:          "nqn.2022-09.io.spdk:opi3",
					SerialNumber: "OpiSerialNumber",
					ModelNumber:  "OpiModelNumber",
				},
			},
			nil,
			`{"id":4,"error":{"code":1,"message":"myopierr"},"result":false}`,
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_create: %v", "json response error: myopierr"),
		},
		{
			"valid request with valid SPDK responce",
			&pb.NVMeSubsystem{
				Spec: &pb.NVMeSubsystemSpec{
					Id:           &pc.ObjectKey{Value: "subsystem-test"},
					Nqn:          "nqn.2022-09.io.spdk:opi3",
					SerialNumber: "OpiSerialNumber",
					ModelNumber:  "OpiModelNumber",
				},
			},
			&pb.NVMeSubsystem{
				Spec: &pb.NVMeSubsystemSpec{
					Id:           &pc.ObjectKey{Value: "subsystem-test"},
					Nqn:          "nqn.2022-09.io.spdk:opi3",
					SerialNumber: "OpiSerialNumber",
					ModelNumber:  "OpiModelNumber",
				},
			},
			`{"id":5,"error":{"code":0,"message":""},"result":true}`, // `{"jsonrpc": "2.0", "id": 1, "result": True}`,
			codes.OK,
			"",
		},
	}

	ctx := context.Background()

	conn, err := grpc.DialContext(ctx, "", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(dialer()))

	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewFrontendNvmeServiceClient(conn)

	// start SPDK mockup server
	if err := os.RemoveAll(*rpcSock); err != nil {
		log.Fatal(err)
	}
	ln, err := net.Listen("unix", *rpcSock)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer ln.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go spdkMockServer(ln, tt.spdk)
			request := &pb.CreateNVMeSubsystemRequest{NvMeSubsystem: tt.in}
			response, err := client.CreateNVMeSubsystem(ctx, request)
			if response != nil {
				if !reflect.DeepEqual(response.Spec, tt.out.Spec) {
					// if response.GetSpec() != tt.out.GetSpec() {
					t.Error("response: expected", tt.out.GetSpec(), "received", response.GetSpec())
				}
			}

			if err != nil {
				if er, ok := status.FromError(err); ok {
					if er.Code() != tt.errCode {
						t.Error("error code: expected", codes.InvalidArgument, "received", er.Code())
					}
					if er.Message() != tt.errMsg {
						t.Error("error message: expected", tt.errMsg, "received", er.Message())
					}
				}
			}
		})
	}
}

func TestFrontEnd_DeleteNVMeSubsystem(t *testing.T) {

}

func TestFrontEnd_UpdateNVMeSubsystem(t *testing.T) {

}

func TestFrontEnd_ListNVMeSubsystem(t *testing.T) {

}

func TestFrontEnd_GetNVMeSubsystem(t *testing.T) {

}

func TestFrontEnd_NVMeSubsystemStats(t *testing.T) {

}
