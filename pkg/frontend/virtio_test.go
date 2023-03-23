// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
)

func TestFrontEnd_CreateVirtioBlk(t *testing.T) {
	tests := map[string]struct {
		in          *pb.VirtioBlk
		out         *pb.VirtioBlk
		spdk        []string
		expectedErr error
	}{
		"valid virtio-blk creation": {
			in:          &testVirtioCtrl,
			out:         &testVirtioCtrl,
			spdk:        []string{`{"id":%d,"error":{"code":0,"message":""},"result":"VblkEmu0pf0"}`},
			expectedErr: status.Error(codes.OK, ""),
		},
		"spdk virtio-blk creation error": {
			in:          &testVirtioCtrl,
			out:         nil,
			spdk:        []string{`{"id":%d,"error":{"code":1,"message":"some internal error"},"result":"VblkEmu0pf0"}`},
			expectedErr: errFailedSpdkCall,
		},
		"spdk virtio-blk creation returned false response with no error": {
			in:          &testVirtioCtrl,
			out:         nil,
			spdk:        []string{`{"id":%d,"error":{"code":0,"message":""},"result":""}`},
			expectedErr: errUnexpectedSpdkCallResult,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			testEnv := createTestEnvironment(true, test.spdk)
			defer testEnv.Close()

			request := &pb.CreateVirtioBlkRequest{VirtioBlk: test.in}
			response, err := testEnv.client.CreateVirtioBlk(testEnv.ctx, request)
			if response != nil {
				wantOut, _ := proto.Marshal(test.out)
				gotOut, _ := proto.Marshal(response)

				if !bytes.Equal(wantOut, gotOut) {
					t.Error("response: expected", test.out, "received", response)
				}
			} else if test.out != nil {
				t.Error("response: expected", test.out, "received nil")
			}

			if err != nil {
				if !strings.Contains(err.Error(), test.expectedErr.Error()) {
					t.Error("expected err contains", test.expectedErr, "received", err)
				}
			} else {
				if test.expectedErr != nil {
					t.Error("expected err contains", test.expectedErr, "received nil")
				}
			}
		})
	}
}

func TestFrontEnd_UpdateVirtioBlk(t *testing.T) {
	tests := []struct {
		name    string
		in      *pb.VirtioBlk
		out     *pb.VirtioBlk
		spdk    []string
		errCode codes.Code
		errMsg  string
		start   bool
	}{
		{
			"unimplemented method",
			&pb.VirtioBlk{},
			nil,
			[]string{""},
			codes.Unimplemented,
			fmt.Sprintf("%v method is not implemented", "UpdateVirtioBlk"),
			false,
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.start, tt.spdk)
			defer testEnv.Close()

			request := &pb.UpdateVirtioBlkRequest{VirtioBlk: tt.in}
			response, err := testEnv.client.UpdateVirtioBlk(testEnv.ctx, request)
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

func TestFrontEnd_ListVirtioBlks(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		out     []*pb.VirtioBlk
		spdk    []string
		errCode codes.Code
		errMsg  string
		start   bool
	}{
		{
			"valid request with invalid SPDK response",
			"subsystem-test",
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			codes.InvalidArgument,
			fmt.Sprintf("Could not create NQN: %v", "nqn.2022-09.io.spdk:opi3"),
			true,
		},
		{
			"valid request with empty SPDK response",
			"subsystem-test",
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("controller_list: %v", "EOF"),
			true,
		},
		{
			"valid request with ID mismatch SPDK response",
			"subsystem-test",
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("controller_list: %v", "json response ID mismatch"),
			true,
		},
		{
			"valid request with error code from SPDK response",
			"subsystem-test",
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("controller_list: %v", "json response error: myopierr"),
			true,
		},
		{
			"valid request with valid SPDK response",
			"subsystem-test",
			[]*pb.VirtioBlk{
				{
					Id:       &pc.ObjectKey{Value: "VblkEmu0pf0"},
					PcieId:   &pb.PciEndpoint{PhysicalFunction: int32(0)},
					VolumeId: &pc.ObjectKey{Value: "TBD"},
				},
				{
					Id:       &pc.ObjectKey{Value: "virtio-blk-42"},
					PcieId:   &pb.PciEndpoint{PhysicalFunction: int32(0)},
					VolumeId: &pc.ObjectKey{Value: "TBD"},
				},
				{
					Id:       &pc.ObjectKey{Value: "VblkEmu0pf2"},
					PcieId:   &pb.PciEndpoint{PhysicalFunction: int32(0)},
					VolumeId: &pc.ObjectKey{Value: "TBD"},
				},
				{},
			},
			[]string{`{"jsonrpc":"2.0","id":%d,"result":[{"name":"VblkEmu0pf0","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"virtio-blk-42","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"VblkEmu0pf2","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"subnqn":"nqn.2020-12.mlnx.snap","cntlid":0,"name":"NvmeEmu0pf0","emulation_manager":"mlx5_0","type":"nvme","pci_index":0,"pci_bdf":"ca:00.2"}],"error":{"code":0,"message":""}}`},
			codes.OK,
			"",
			true,
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.start, tt.spdk)
			defer testEnv.Close()

			request := &pb.ListVirtioBlksRequest{Parent: tt.in}
			response, err := testEnv.client.ListVirtioBlks(testEnv.ctx, request)

			if response != nil {
				if !reflect.DeepEqual(response.VirtioBlks, tt.out) {
					t.Error("response: expected", tt.out, "received", response.VirtioBlks)
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

func TestFrontEnd_GetVirtioBlk(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		out     *pb.VirtioBlk
		spdk    []string
		errCode codes.Code
		errMsg  string
		start   bool
	}{
		{
			"valid request with invalid SPDK response",
			"virtio-blk-42",
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			codes.InvalidArgument,
			fmt.Sprintf("Could not find Controller: %v", "virtio-blk-42"),
			true,
		},
		{
			"valid request with empty SPDK response",
			"virtio-blk-42",
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("controller_list: %v", "EOF"),
			true,
		},
		{
			"valid request with ID mismatch SPDK response",
			"virtio-blk-42",
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("controller_list: %v", "json response ID mismatch"),
			true,
		},
		{
			"valid request with error code from SPDK response",
			"virtio-blk-42",
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("controller_list: %v", "json response error: myopierr"),
			true,
		},
		{
			"valid request with valid SPDK response",
			"virtio-blk-42",
			&pb.VirtioBlk{
				Id:       &pc.ObjectKey{Value: "virtio-blk-42"},
				PcieId:   &pb.PciEndpoint{PhysicalFunction: int32(0)},
				VolumeId: &pc.ObjectKey{Value: "TBD"},
			},
			[]string{`{"jsonrpc":"2.0","id":%d,"result":[{"name":"VblkEmu0pf0","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"virtio-blk-42","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"VblkEmu0pf2","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"subnqn":"nqn.2020-12.mlnx.snap","cntlid":0,"name":"NvmeEmu0pf0","emulation_manager":"mlx5_0","type":"nvme","pci_index":0,"pci_bdf":"ca:00.2"}],"error":{"code":0,"message":""}}`},
			codes.OK,
			"",
			true,
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.start, tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.VirtioCtrls[testVirtioCtrl.Id.Value] = &testVirtioCtrl

			request := &pb.GetVirtioBlkRequest{Name: tt.in}
			response, err := testEnv.client.GetVirtioBlk(testEnv.ctx, request)
			if response != nil {
				wantOut, _ := proto.Marshal(tt.out)
				gotOut, _ := proto.Marshal(response)
				if !bytes.Equal(wantOut, gotOut) {
					t.Error("response: expected", tt.out, "received", response)
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

func TestFrontEnd_VirtioBlkStats(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		out     *pb.VolumeStats
		spdk    []string
		errCode codes.Code
		errMsg  string
		start   bool
	}{
		{
			"valid request with invalid SPDK response",
			"namespace-test",
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":{"controllers":[{"name":"NvmeEmu0pf1","bdevs":[]}]}}`},
			codes.InvalidArgument,
			fmt.Sprintf("Could not find Controller: %v", "namespace-test"),
			true,
		},
		{
			"valid request with invalid marshal SPDK response",
			"namespace-test",
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("controller_virtio_blk_get_iostat: %v", "json: cannot unmarshal array into Go value of type models.NvdaControllerNvmeStatsResult"),
			true,
		},
		{
			"valid request with empty SPDK response",
			"namespace-test",
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("controller_virtio_blk_get_iostat: %v", "EOF"),
			true,
		},
		{
			"valid request with ID mismatch SPDK response",
			"namespace-test",
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":{"controllers":[{"name":"NvmeEmu0pf1","bdevs":[]}]}}`},
			codes.Unknown,
			fmt.Sprintf("controller_virtio_blk_get_iostat: %v", "json response ID mismatch"),
			true,
		},
		{
			"valid request with error code from SPDK response",
			"namespace-test",
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"}}`},
			codes.Unknown,
			fmt.Sprintf("controller_virtio_blk_get_iostat: %v", "json response error: myopierr"),
			true,
		},
		{
			"valid request with valid SPDK response",
			"Malloc0",
			&pb.VolumeStats{
				ReadOpsCount:  12345,
				WriteOpsCount: 54321,
			},
			[]string{`{"jsonrpc":"2.0","id":%d,"result":{"controllers":[{"name":"VblkEmu0pf0","bdevs":[{"bdev_name":"Malloc0","read_ios":12345,"completed_read_ios":0,"completed_unordered_read_ios":0,"write_ios":54321,"completed_write_ios":0,"completed_unordered_write_ios":0,"flush_ios":0,"completed_flush_ios":0,"completed_unordered_flush_ios":0,"err_read_ios":0,"err_write_ios":0,"err_flush_ios":0}]}]},"error":{"code":0,"message":""}}`},
			codes.OK,
			"",
			true,
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.start, tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.VirtioCtrls[testVirtioCtrl.Id.Value] = &testVirtioCtrl

			request := &pb.VirtioBlkStatsRequest{ControllerId: &pc.ObjectKey{Value: tt.in}}
			response, err := testEnv.client.VirtioBlkStats(testEnv.ctx, request)
			if response != nil {
				if !reflect.DeepEqual(response.Stats, tt.out) {
					t.Error("response: expected", tt.out, "received", response.Stats)
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

func TestFrontEnd_DeleteVirtioBlk(t *testing.T) {
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
			"valid request with invalid SPDK response",
			"virtio-blk-42",
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			codes.InvalidArgument,
			fmt.Sprintf("Could not delete NQN:ID %v", "nqn.2022-09.io.spdk:opi3:17"),
			true,
		},
		{
			"valid request with empty SPDK response",
			"virtio-blk-42",
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("controller_virtio_blk_delete: %v", "EOF"),
			true,
		},
		{
			"valid request with ID mismatch SPDK response",
			"virtio-blk-42",
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":false}`},
			codes.Unknown,
			fmt.Sprintf("controller_virtio_blk_delete: %v", "json response ID mismatch"),
			true,
		},
		{
			"valid request with error code from SPDK response",
			"virtio-blk-42",
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":false}`},
			codes.Unknown,
			fmt.Sprintf("controller_virtio_blk_delete: %v", "json response error: myopierr"),
			true,
		},
		{
			"valid request with valid SPDK response",
			"virtio-blk-42",
			&emptypb.Empty{},
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`}, // `{"jsonrpc": "2.0", "id": 1, "result": True}`,
			codes.OK,
			"",
			true,
		},
	}

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.start, tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.VirtioCtrls[testVirtioCtrl.Id.Value] = &testVirtioCtrl

			request := &pb.DeleteVirtioBlkRequest{Name: tt.in}
			response, err := testEnv.client.DeleteVirtioBlk(testEnv.ctx, request)
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
