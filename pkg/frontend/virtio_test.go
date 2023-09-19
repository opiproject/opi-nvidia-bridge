// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// Copyright (C) 2023 Intel Corporation

// Package frontend implememnts the FrontEnd APIs (host facing) of the storage Server
package frontend

import (
	"fmt"
	"reflect"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/server"
)

func TestFrontEnd_CreateVirtioBlk(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      *pb.VirtioBlk
		out     *pb.VirtioBlk
		spdk    []string
		errCode codes.Code
		errMsg  string
		exist   bool
	}{
		"valid virtio-blk creation": {
			in:      &testVirtioCtrl,
			out:     &testVirtioCtrl,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":"VblkEmu0pf0"}`},
			errCode: codes.OK,
			errMsg:  "",
		},
		"spdk virtio-blk creation error": {
			in:      &testVirtioCtrl,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"some internal error"},"result":"VblkEmu0pf0"}`},
			errCode: codes.Unknown,
			errMsg:  "controller_virtio_blk_create: json response error: some internal error",
		},
		"spdk virtio-blk creation returned false response with no error": {
			in:      &testVirtioCtrl,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":""}`},
			errCode: codes.InvalidArgument,
			errMsg:  fmt.Sprintf("Could not create virtio-blk: %s", testVirtioCtrlID),
		},
		"no required field": {
			in:      nil,
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: virtio_blk",
		},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			if tt.out != nil {
				tt.out = server.ProtoClone(tt.out)
				tt.out.Name = testVirtioCtrlName
			}

			request := &pb.CreateVirtioBlkRequest{VirtioBlk: tt.in, VirtioBlkId: testVirtioCtrlID}
			response, err := testEnv.client.CreateVirtioBlk(testEnv.ctx, request)

			if !proto.Equal(response, tt.out) {
				t.Error("response: expected", tt.out, "received", response)
			}

			if er, ok := status.FromError(err); ok {
				if er.Code() != tt.errCode {
					t.Error("error code: expected", tt.errCode, "received", er.Code())
				}
				if er.Message() != tt.errMsg {
					t.Error("error message: expected", tt.errMsg, "received", er.Message())
				}
			} else {
				t.Error("expected grpc error status")
			}
		})
	}
}

func TestFrontEnd_UpdateVirtioBlk(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		mask    *fieldmaskpb.FieldMask
		in      *pb.VirtioBlk
		out     *pb.VirtioBlk
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"invalid fieldmask": {
			mask: &fieldmaskpb.FieldMask{Paths: []string{"*", "author"}},
			in: &pb.VirtioBlk{
				Name:          testVirtioCtrlName,
				PcieId:        testVirtioCtrl.PcieId,
				VolumeNameRef: testVirtioCtrl.VolumeNameRef,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("invalid field path: %s", "'*' must not be used with other paths"),
		},
		"no required field": {
			mask:    nil,
			in:      nil,
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: virtio_blk",
		},
		"no required pcie field": {
			mask: nil,
			in: &pb.VirtioBlk{
				Name: testVirtioCtrlName,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: virtio_blk.pcie_id",
		},
		"no required volume field": {
			mask: nil,
			in: &pb.VirtioBlk{
				Name:   testVirtioCtrlName,
				PcieId: testVirtioCtrl.PcieId,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: virtio_blk.volume_name_ref",
		},
		"unimplemented method": {
			mask: nil,
			in: &pb.VirtioBlk{
				Name:          testVirtioCtrlName,
				PcieId:        testVirtioCtrl.PcieId,
				VolumeNameRef: testVirtioCtrl.VolumeNameRef,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unimplemented,
			errMsg:  fmt.Sprintf("%v method is not implemented", "UpdateVirtioBlk"),
		},
		"valid request with unknown key": {
			mask: nil,
			in: &pb.VirtioBlk{
				Name: server.ResourceIDToVolumeName("unknown-id"),
				PcieId: &pb.PciEndpoint{
					PhysicalFunction: wrapperspb.Int32(42),
					VirtualFunction:  wrapperspb.Int32(0),
					PortId:           wrapperspb.Int32(0),
				},
				VolumeNameRef: "Malloc42",
				MaxIoQps:      1,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find key %v", server.ResourceIDToVolumeName("unknown-id")),
		},
		"malformed name": {
			mask: nil,
			in: &pb.VirtioBlk{
				Name:          "-ABC-DEF",
				PcieId:        testVirtioCtrl.PcieId,
				VolumeNameRef: testVirtioCtrl.VolumeNameRef,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("segment '%s': not a valid DNS name", "-ABC-DEF"),
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.VirtioCtrls[testVirtioCtrlName] = server.ProtoClone(&testVirtioCtrl)
			testEnv.opiSpdkServer.VirtioCtrls[testVirtioCtrlName].Name = testVirtioCtrlName

			request := &pb.UpdateVirtioBlkRequest{VirtioBlk: tt.in, UpdateMask: tt.mask}
			response, err := testEnv.client.UpdateVirtioBlk(testEnv.ctx, request)

			if !proto.Equal(response, tt.out) {
				t.Error("response: expected", tt.out, "received", response)
			}

			if er, ok := status.FromError(err); ok {
				if er.Code() != tt.errCode {
					t.Error("error code: expected", tt.errCode, "received", er.Code())
				}
				if er.Message() != tt.errMsg {
					t.Error("error message: expected", tt.errMsg, "received", er.Message())
				}
			} else {
				t.Error("expected grpc error status")
			}
		})
	}
}

func TestFrontEnd_ListVirtioBlks(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     []*pb.VirtioBlk
		spdk    []string
		errCode codes.Code
		errMsg  string
		size    int32
		token   string
	}{
		"valid request with empty empty result SPDK response": {
			in:      "subsystem-test",
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    0,
			token:   "",
		},
		"valid request with empty SPDK response": {
			in:      "subsystem-test",
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "EOF"),
			size:    0,
			token:   "",
		},
		"valid request with ID mismatch SPDK response": {
			in:      "subsystem-test",
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "json response ID mismatch"),
			size:    0,
			token:   "",
		},
		"valid request with error code from SPDK response": {
			in:      "subsystem-test",
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "json response error: myopierr"),
			size:    0,
			token:   "",
		},
		"pagination negative": {
			in:      "subsystem-test",
			out:     nil,
			spdk:    []string{},
			errCode: codes.InvalidArgument,
			errMsg:  "negative PageSize is not allowed",
			size:    -10,
			token:   "",
		},
		"pagination error": {
			in:      "subsystem-test",
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find pagination token %s", "unknown-pagination-token"),
			size:    0,
			token:   "unknown-pagination-token",
		},
		"pagination": {
			in: "subsystem-test",
			out: []*pb.VirtioBlk{
				{
					Name: server.ResourceIDToVolumeName("VblkEmu0pf0"),
					PcieId: &pb.PciEndpoint{
						PhysicalFunction: wrapperspb.Int32(0),
						VirtualFunction:  wrapperspb.Int32(0),
						PortId:           wrapperspb.Int32(0),
					},
					VolumeNameRef: "TBD",
				},
			},
			spdk:    []string{`{"jsonrpc":"2.0","id":%d,"result":[{"name":"VblkEmu0pf0","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"virtio-blk-42","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"VblkEmu0pf2","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"subnqn":"nqn.2020-12.mlnx.snap","cntlid":0,"name":"NvmeEmu0pf0","emulation_manager":"mlx5_0","type":"nvme","pci_index":0,"pci_bdf":"ca:00.2"}],"error":{"code":0,"message":""}}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    1,
			token:   "",
		},
		"pagination overflow": {
			in: "subsystem-test",
			out: []*pb.VirtioBlk{
				{
					Name: server.ResourceIDToVolumeName("VblkEmu0pf0"),
					PcieId: &pb.PciEndpoint{
						PhysicalFunction: wrapperspb.Int32(0),
						VirtualFunction:  wrapperspb.Int32(0),
						PortId:           wrapperspb.Int32(0),
					},
					VolumeNameRef: "TBD",
				},
				{
					Name: server.ResourceIDToVolumeName("VblkEmu0pf2"),
					PcieId: &pb.PciEndpoint{
						PhysicalFunction: wrapperspb.Int32(0),
						VirtualFunction:  wrapperspb.Int32(0),
						PortId:           wrapperspb.Int32(0),
					},
					VolumeNameRef: "TBD",
				},
				{
					Name: testVirtioCtrlName,
					PcieId: &pb.PciEndpoint{
						PhysicalFunction: wrapperspb.Int32(0),
						VirtualFunction:  wrapperspb.Int32(0),
						PortId:           wrapperspb.Int32(0),
					},
					VolumeNameRef: "TBD",
				},
			},
			spdk:    []string{`{"jsonrpc":"2.0","id":%d,"result":[{"name":"VblkEmu0pf0","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"virtio-blk-42","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"VblkEmu0pf2","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"subnqn":"nqn.2020-12.mlnx.snap","cntlid":0,"name":"NvmeEmu0pf0","emulation_manager":"mlx5_0","type":"nvme","pci_index":0,"pci_bdf":"ca:00.2"}],"error":{"code":0,"message":""}}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    1000,
			token:   "",
		},
		"pagination offset": {
			in: "subsystem-test",
			out: []*pb.VirtioBlk{
				{
					Name: testVirtioCtrlName,
					PcieId: &pb.PciEndpoint{
						PhysicalFunction: wrapperspb.Int32(0),
						VirtualFunction:  wrapperspb.Int32(0),
						PortId:           wrapperspb.Int32(0),
					},
					VolumeNameRef: "TBD",
				},
			},
			spdk:    []string{`{"jsonrpc":"2.0","id":%d,"result":[{"name":"VblkEmu0pf0","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"virtio-blk-42","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"VblkEmu0pf2","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"subnqn":"nqn.2020-12.mlnx.snap","cntlid":0,"name":"NvmeEmu0pf0","emulation_manager":"mlx5_0","type":"nvme","pci_index":0,"pci_bdf":"ca:00.2"}],"error":{"code":0,"message":""}}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    1,
			token:   "existing-pagination-token",
		},
		"valid request with valid SPDK response": {
			in: "subsystem-test",
			out: []*pb.VirtioBlk{
				{
					Name: server.ResourceIDToVolumeName("VblkEmu0pf0"),
					PcieId: &pb.PciEndpoint{
						PhysicalFunction: wrapperspb.Int32(0),
						VirtualFunction:  wrapperspb.Int32(0),
						PortId:           wrapperspb.Int32(0),
					},
					VolumeNameRef: "TBD",
				},
				{
					Name: server.ResourceIDToVolumeName("VblkEmu0pf2"),
					PcieId: &pb.PciEndpoint{
						PhysicalFunction: wrapperspb.Int32(0),
						VirtualFunction:  wrapperspb.Int32(0),
						PortId:           wrapperspb.Int32(0),
					},
					VolumeNameRef: "TBD",
				},
				{
					Name: testVirtioCtrlName,
					PcieId: &pb.PciEndpoint{
						PhysicalFunction: wrapperspb.Int32(0),
						VirtualFunction:  wrapperspb.Int32(0),
						PortId:           wrapperspb.Int32(0),
					},
					VolumeNameRef: "TBD",
				},
			},
			spdk:    []string{`{"jsonrpc":"2.0","id":%d,"result":[{"name":"VblkEmu0pf0","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"virtio-blk-42","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"VblkEmu0pf2","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"subnqn":"nqn.2020-12.mlnx.snap","cntlid":0,"name":"NvmeEmu0pf0","emulation_manager":"mlx5_0","type":"nvme","pci_index":0,"pci_bdf":"ca:00.2"}],"error":{"code":0,"message":""}}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    0,
			token:   "",
		},
		"no required field": {
			in:      "",
			out:     []*pb.VirtioBlk{},
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: parent",
			size:    0,
			token:   "",
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.Pagination["existing-pagination-token"] = 1

			request := &pb.ListVirtioBlksRequest{Parent: tt.in, PageSize: tt.size, PageToken: tt.token}
			response, err := testEnv.client.ListVirtioBlks(testEnv.ctx, request)

			if !server.EqualProtoSlices(response.GetVirtioBlks(), tt.out) {
				t.Error("response: expected", tt.out, "received", response.GetVirtioBlks())
			}

			// Empty NextPageToken indicates end of results list
			if tt.size != 1 && response.GetNextPageToken() != "" {
				t.Error("Expected end of results, received non-empty next page token", response.GetNextPageToken())
			}

			if er, ok := status.FromError(err); ok {
				if er.Code() != tt.errCode {
					t.Error("error code: expected", tt.errCode, "received", er.Code())
				}
				if er.Message() != tt.errMsg {
					t.Error("error message: expected", tt.errMsg, "received", er.Message())
				}
			} else {
				t.Error("expected grpc error status")
			}
		})
	}
}

func TestFrontEnd_GetVirtioBlk(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     *pb.VirtioBlk
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"valid request with empty result SPDK response": {
			in:      testVirtioCtrlName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.InvalidArgument,
			errMsg:  fmt.Sprintf("Could not find Controller: %v", testVirtioCtrlName),
		},
		"valid request with empty SPDK response": {
			in:      testVirtioCtrlName,
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "EOF"),
		},
		"valid request with ID mismatch SPDK response": {
			in:      testVirtioCtrlName,
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "json response ID mismatch"),
		},
		"valid request with error code from SPDK response": {
			in:      testVirtioCtrlName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "json response error: myopierr"),
		},
		"valid request with valid SPDK response": {
			in: testVirtioCtrlName,
			out: &pb.VirtioBlk{
				Name: testVirtioCtrlName,
				PcieId: &pb.PciEndpoint{
					PhysicalFunction: wrapperspb.Int32(0),
					VirtualFunction:  wrapperspb.Int32(0),
					PortId:           wrapperspb.Int32(0),
				},
				VolumeNameRef: "TBD",
			},
			spdk:    []string{`{"jsonrpc":"2.0","id":%d,"result":[{"name":"VblkEmu0pf0","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"virtio-blk-42","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"name":"VblkEmu0pf2","emulation_manager":"mlx5_0","type":"virtio_blk","pci_index":0,"pci_bdf":"ca:00.4"},{"subnqn":"nqn.2020-12.mlnx.snap","cntlid":0,"name":"NvmeEmu0pf0","emulation_manager":"mlx5_0","type":"nvme","pci_index":0,"pci_bdf":"ca:00.2"}],"error":{"code":0,"message":""}}`},
			errCode: codes.OK,
			errMsg:  "",
		},
		"malformed name": {
			in:      "-ABC-DEF",
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("segment '%s': not a valid DNS name", "-ABC-DEF"),
		},
		"no required field": {
			in:      "",
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: name",
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.VirtioCtrls[testVirtioCtrlName] = server.ProtoClone(&testVirtioCtrl)

			request := &pb.GetVirtioBlkRequest{Name: tt.in}
			response, err := testEnv.client.GetVirtioBlk(testEnv.ctx, request)

			if !proto.Equal(response, tt.out) {
				t.Error("response: expected", tt.out, "received", response)
			}

			if er, ok := status.FromError(err); ok {
				if er.Code() != tt.errCode {
					t.Error("error code: expected", tt.errCode, "received", er.Code())
				}
				if er.Message() != tt.errMsg {
					t.Error("error message: expected", tt.errMsg, "received", er.Message())
				}
			} else {
				t.Error("expected grpc error status")
			}
		})
	}
}

func TestFrontEnd_VirtioBlkStats(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     *pb.VolumeStats
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"valid request with invalid SPDK response": {
			in:      "namespace-test",
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"controllers":[{"name":"NvmeEmu0pf1","bdevs":[]}]}}`},
			errCode: codes.InvalidArgument,
			errMsg:  fmt.Sprintf("Could not find Controller: %v", "namespace-test"),
		},
		"valid request with invalid marshal SPDK response": {
			in:      "namespace-test",
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_virtio_blk_get_iostat: %v", "json: cannot unmarshal array into Go value of type models.NvdaControllerNvmeStatsResult"),
		},
		"valid request with empty SPDK response": {
			in:      "namespace-test",
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_virtio_blk_get_iostat: %v", "EOF"),
		},
		"valid request with ID mismatch SPDK response": {
			in:      "namespace-test",
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":{"controllers":[{"name":"NvmeEmu0pf1","bdevs":[]}]}}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_virtio_blk_get_iostat: %v", "json response ID mismatch"),
		},
		"valid request with error code from SPDK response": {
			in:      "namespace-test",
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"}}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_virtio_blk_get_iostat: %v", "json response error: myopierr"),
		},
		"valid request with valid SPDK response": {
			in: "Malloc0",
			out: &pb.VolumeStats{
				ReadOpsCount:  12345,
				WriteOpsCount: 54321,
			},
			spdk:    []string{`{"jsonrpc":"2.0","id":%d,"result":{"controllers":[{"name":"VblkEmu0pf0","bdevs":[{"bdev_name":"Malloc0","read_ios":12345,"completed_read_ios":0,"completed_unordered_read_ios":0,"write_ios":54321,"completed_write_ios":0,"completed_unordered_write_ios":0,"flush_ios":0,"completed_flush_ios":0,"completed_unordered_flush_ios":0,"err_read_ios":0,"err_write_ios":0,"err_flush_ios":0}]}]},"error":{"code":0,"message":""}}`},
			errCode: codes.OK,
			errMsg:  "",
		},
		"malformed name": {
			in:      "-ABC-DEF",
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("segment '%s': not a valid DNS name", "-ABC-DEF"),
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.VirtioCtrls[testVirtioCtrl.Name] = server.ProtoClone(&testVirtioCtrl)

			request := &pb.StatsVirtioBlkRequest{Name: tt.in}
			response, err := testEnv.client.StatsVirtioBlk(testEnv.ctx, request)

			if !proto.Equal(response.GetStats(), tt.out) {
				t.Error("response: expected", tt.out, "received", response.GetStats())
			}

			if er, ok := status.FromError(err); ok {
				if er.Code() != tt.errCode {
					t.Error("error code: expected", tt.errCode, "received", er.Code())
				}
				if er.Message() != tt.errMsg {
					t.Error("error message: expected", tt.errMsg, "received", er.Message())
				}
			} else {
				t.Error("expected grpc error status")
			}
		})
	}
}

func TestFrontEnd_DeleteVirtioBlk(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     *emptypb.Empty
		spdk    []string
		errCode codes.Code
		errMsg  string
		missing bool
	}{
		"valid request with false as result SPDK response": {
			in:      testVirtioCtrlName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			errCode: codes.OK,
			errMsg:  "",
			missing: false,
		},
		"valid request with empty SPDK response": {
			in:      testVirtioCtrlName,
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_virtio_blk_delete: %v", "EOF"),
			missing: false,
		},
		"valid request with ID mismatch SPDK response": {
			in:      testVirtioCtrlName,
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":false}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_virtio_blk_delete: %v", "json response ID mismatch"),
			missing: false,
		},
		"valid request with error code from SPDK response": {
			in:      testVirtioCtrlName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":false}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_virtio_blk_delete: %v", "json response error: myopierr"),
			missing: false,
		},
		"valid request with valid SPDK response": {
			in:      testVirtioCtrlName,
			out:     &emptypb.Empty{},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`}, // `{"jsonrpc": "2.0", "id": 1, "result": True}`,
			errCode: codes.OK,
			errMsg:  "",
			missing: false,
		},
		"unknown key with missing allowed": {
			in:      server.ResourceIDToVolumeName("unknown-id"),
			out:     &emptypb.Empty{},
			spdk:    []string{},
			errCode: codes.OK,
			errMsg:  "",
			missing: true,
		},
		"malformed name": {
			in:      "-ABC-DEF",
			out:     &emptypb.Empty{},
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("segment '%s': not a valid DNS name", "-ABC-DEF"),
			missing: false,
		},
		"no required field": {
			in:      "",
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: name",
			missing: false,
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.VirtioCtrls[testVirtioCtrlName] = server.ProtoClone(&testVirtioCtrl)

			request := &pb.DeleteVirtioBlkRequest{Name: tt.in, AllowMissing: tt.missing}
			response, err := testEnv.client.DeleteVirtioBlk(testEnv.ctx, request)

			if er, ok := status.FromError(err); ok {
				if er.Code() != tt.errCode {
					t.Error("error code: expected", tt.errCode, "received", er.Code())
				}
				if er.Message() != tt.errMsg {
					t.Error("error message: expected", tt.errMsg, "received", er.Message())
				}
			} else {
				t.Error("expected grpc error status")
			}

			if reflect.TypeOf(response) != reflect.TypeOf(tt.out) {
				t.Error("response: expected", reflect.TypeOf(tt.out), "received", reflect.TypeOf(response))
			}
		})
	}
}
