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
	"strings"
	"reflect"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

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

func spdkMockServer(l net.Listener, toSend []string) {
	for _, spdk := range toSend {
		fd, err := l.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		log.Printf("SPDK mockup server: client connected [%s]", fd.RemoteAddr().Network())
		log.Printf("SPDK ID [%d]", rpcID)

		buf := make([]byte, 512)
		nr, err := fd.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]
		if strings.Contains(spdk, "%") {
			spdk = fmt.Sprintf(spdk, rpcID)
		}

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
}

func TestFrontEnd_CreateNVMeSubsystem(t *testing.T) {
	tests := []struct {
		name    string
		in      *pb.NVMeSubsystem
		out     *pb.NVMeSubsystem
		spdk    []string
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
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
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
			[]string{""},
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
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":false}`},
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
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":false}`},
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
				Status: &pb.NVMeSubsystemStatus{
					FirmwareRevision: "SPDK v20.10",
				},
			},
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`, `{"jsonrpc":"2.0","id":%d,"result":{"version":"SPDK v20.10","fields":{"major":20,"minor":10,"patch":0,"suffix":""}}}`}, // `{"jsonrpc": "2.0", "id": 1, "result": True}`,
			codes.OK,
			"",
		},
	}

	// start GRPC mockup server
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

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go spdkMockServer(ln, tt.spdk)
			request := &pb.CreateNVMeSubsystemRequest{NvMeSubsystem: tt.in}
			response, err := client.CreateNVMeSubsystem(ctx, request)
			if response != nil {
				if !reflect.DeepEqual(response.Spec, tt.out.Spec) {
					t.Error("response: expected", tt.out.GetSpec(), "received", response.GetSpec())
				}
				if !reflect.DeepEqual(response.Status, tt.out.Status) {
					t.Error("response: expected", tt.out.GetStatus(), "received", response.GetStatus())
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

func TestFrontEnd_UpdateNVMeSubsystem(t *testing.T) {
	tests := []struct {
		name    string
		in      *pb.NVMeSubsystem
		out     *pb.NVMeSubsystem
		errCode codes.Code
		errMsg  string
	}{
		{
			"unimplemented method",
			&pb.NVMeSubsystem{},
			nil,
			codes.Unimplemented,
			fmt.Sprintf("%v method is not implemented", "UpdateNVMeSubsystem"),
		},
	}

	// start GRPC mockup server
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(dialer()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := pb.NewFrontendNvmeServiceClient(conn)

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &pb.UpdateNVMeSubsystemRequest{NvMeSubsystem: tt.in}
			response, err := client.UpdateNVMeSubsystem(ctx, request)
			if response != nil {
				t.Error("response: expected", codes.Unimplemented, "received", response)
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

func TestFrontEnd_ListNVMeSubsystem(t *testing.T) {
	tests := []struct {
		name    string
		out     []*pb.NVMeSubsystem
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		{
			"valid request with invalid SPDK responce",
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			codes.InvalidArgument,
			fmt.Sprintf("Could not create NQN: %v", "nqn.2022-09.io.spdk:opi3"),
		},
		{
			"valid request with empty SPDK responce",
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "EOF"),
		},
		{
			"valid request with ID mismatch SPDK responce",
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "json response ID mismatch"),
		},
		{
			"valid request with error code from SPDK responce",
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "json response error: myopierr"),
		},
		{
			"valid request with valid SPDK responce",
			[]*pb.NVMeSubsystem{
				{
					Spec: &pb.NVMeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi1",
						SerialNumber: "OpiSerialNumber1",
						ModelNumber:  "OpiModelNumber1",
					},
				},
				{
					Spec: &pb.NVMeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi2",
						SerialNumber: "OpiSerialNumber2",
						ModelNumber:  "OpiModelNumber2",
					},
				},
				{
					Spec: &pb.NVMeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi3",
						SerialNumber: "OpiSerialNumber3",
						ModelNumber:  "OpiModelNumber3",
					},
				},
			},
			// {'jsonrpc': '2.0', 'id': 1, 'result': [{'nqn': 'nqn.2020-12.mlnx.snap', 'serial_number': 'Mellanox_NVMe_SNAP', 'model_number': 'Mellanox NVMe SNAP Controller', 'controllers': [{'name': 'NvmeEmu0pf1', 'cntlid': 0, 'pci_bdf': 'ca:00.3', 'pci_index': 1}]}]}
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"nqn": "nqn.2022-09.io.spdk:opi1", "serial_number": "OpiSerialNumber1", "model_number": "OpiModelNumber1"},{"nqn": "nqn.2022-09.io.spdk:opi2", "serial_number": "OpiSerialNumber2", "model_number": "OpiModelNumber2"},{"nqn": "nqn.2022-09.io.spdk:opi3", "serial_number": "OpiSerialNumber3", "model_number": "OpiModelNumber3"}]}`},
			codes.OK,
			"",
		},
	}

	// start GRPC mockup server
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

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go spdkMockServer(ln, tt.spdk)
			request := &pb.ListNVMeSubsystemsRequest{}
			response, err := client.ListNVMeSubsystems(ctx, request)
			if response != nil {
				if !reflect.DeepEqual(response.NvMeSubsystems, tt.out) {
					t.Error("response: expected", tt.out, "received", response.NvMeSubsystems)
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

func TestFrontEnd_GetNVMeSubsystem(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		out     *pb.NVMeSubsystem
		spdk    []string
		errCode codes.Code
		errMsg  string
		start   bool
	}{
		{
			"valid request with invalid SPDK responce",
			"subsystem-test",
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			codes.InvalidArgument,
			fmt.Sprintf("Could not find NQN: %v", "nqn.2022-09.io.spdk:opi3"),
			true,
		},
		{
			"valid request with empty SPDK responce",
			"subsystem-test",
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "EOF"),
			true,
		},
		{
			"valid request with ID mismatch SPDK responce",
			"subsystem-test",
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "json response ID mismatch"),
			true,
		},
		{
			"valid request with error code from SPDK responce",
			"subsystem-test",
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "json response error: myopierr"),
			true,
		},
		{
			"valid request with valid SPDK responce",
			"subsystem-test",
			&pb.NVMeSubsystem{
				Spec: &pb.NVMeSubsystemSpec{
					Nqn:          "nqn.2022-09.io.spdk:opi3",
					SerialNumber: "OpiSerialNumber3",
					ModelNumber:  "OpiModelNumber3",
				},
			},
			// {'jsonrpc': '2.0', 'id': 1, 'result': [{'nqn': 'nqn.2020-12.mlnx.snap', 'serial_number': 'Mellanox_NVMe_SNAP', 'model_number': 'Mellanox NVMe SNAP Controller', 'controllers': [{'name': 'NvmeEmu0pf1', 'cntlid': 0, 'pci_bdf': 'ca:00.3', 'pci_index': 1}]}]}
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"nqn": "nqn.2022-09.io.spdk:opi1", "serial_number": "OpiSerialNumber1", "model_number": "OpiModelNumber1"},{"nqn": "nqn.2022-09.io.spdk:opi2", "serial_number": "OpiSerialNumber2", "model_number": "OpiModelNumber2"},{"nqn": "nqn.2022-09.io.spdk:opi3", "serial_number": "OpiSerialNumber3", "model_number": "OpiModelNumber3"}]}`},
			codes.OK,
			"",
			true,
		},
		{
			"valid request with unknown key",
			"unknown-subsystem-id",
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("unable to find key %v", "unknown-subsystem-id"),
			false,
		},
	}

	// start GRPC mockup server
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

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.start {
				go spdkMockServer(ln, tt.spdk)
			}
			request := &pb.GetNVMeSubsystemRequest{Name: tt.in}
			response, err := client.GetNVMeSubsystem(ctx, request)
			if response != nil {
				if !reflect.DeepEqual(response.Spec, tt.out.Spec) {
					t.Error("response: expected", tt.out.GetSpec(), "received", response.GetSpec())
				}
				// TODO: compare more Status fields
				if response.Status.FirmwareRevision != "TBD" {
					t.Error("response: expected", tt.out.GetStatus(), "received", response.GetStatus())
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

func TestFrontEnd_NVMeSubsystemStats(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		out     *pb.NVMeSubsystemStatsResponse
		errCode codes.Code
		errMsg  string
	}{
		{
			"unimplemented method",
			"subsystem-test",
			nil,
			codes.Unimplemented,
			fmt.Sprintf("%v method is not implemented", "UpdateNVMeSubsystem"),
		},
	}

	// start GRPC mockup server
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(dialer()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := pb.NewFrontendNvmeServiceClient(conn)

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &pb.NVMeSubsystemStatsRequest{SubsystemId: &pc.ObjectKey{Value: tt.in}}
			response, err := client.NVMeSubsystemStats(ctx, request)
			if response != nil {
				t.Error("response: expected", codes.Unimplemented, "received", response)
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
	tests := []struct {
		name    string
		in      string
		out     *emptypb.Empty
		spdk    []string
		errCode codes.Code
		errMsg  string
		start   bool
	}{
		{
			"valid request with invalid SPDK responce",
			"subsystem-test",
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			codes.InvalidArgument,
			fmt.Sprintf("Could not delete NQN: %v", "nqn.2022-09.io.spdk:opi3"),
			true,
		},
		{
			"valid request with empty SPDK responce",
			"subsystem-test",
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_delete: %v", "EOF"),
			true,
		},
		{
			"valid request with ID mismatch SPDK responce",
			"subsystem-test",
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":false}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_delete: %v", "json response ID mismatch"),
			true,
		},
		{
			"valid request with error code from SPDK responce",
			"subsystem-test",
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":false}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_delete: %v", "json response error: myopierr"),
			true,
		},
		{
			"valid request with valid SPDK responce",
			"subsystem-test",
			&emptypb.Empty{},
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`}, // `{"jsonrpc": "2.0", "id": 1, "result": True}`,
			codes.OK,
			"",
			true,
		},
		{
			"valid request with unknown key",
			"unknown-subsystem-id",
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("unable to find key %v", "unknown-subsystem-id"),
			false,
		},
	}

	// start GRPC mockup server
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

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.start {
				go spdkMockServer(ln, tt.spdk)
			}
			request := &pb.DeleteNVMeSubsystemRequest{Name: tt.in}
			response, err := client.DeleteNVMeSubsystem(ctx, request)
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
			if reflect.TypeOf(response) != reflect.TypeOf(tt.out) {
				t.Error("response: expected", reflect.TypeOf(tt.out), "received", reflect.TypeOf(response))
			}
		})
	}
}
