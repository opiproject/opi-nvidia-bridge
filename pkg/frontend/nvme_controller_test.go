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

	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/frontend"
	"github.com/opiproject/opi-spdk-bridge/pkg/utils"
)

func TestFrontEnd_CreateNvmeController(t *testing.T) {
	spec := &pb.NvmeControllerSpec{
		Endpoint:         testController.Spec.Endpoint,
		Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_PCIE,
		NvmeControllerId: proto.Int32(1),
	}
	controllerSpec := &pb.NvmeControllerSpec{
		Endpoint:         testController.Spec.Endpoint,
		Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_PCIE,
		NvmeControllerId: proto.Int32(17),
	}
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	t.Cleanup(utils.CheckTestProtoObjectsNotChanged(spec, controllerSpec)(t, t.Name()))

	tests := map[string]struct {
		id      string
		in      *pb.NvmeController
		out     *pb.NvmeController
		spdk    []string
		errCode codes.Code
		errMsg  string
		exist   bool
		subsys  string
	}{
		"illegal resource_id": {
			id: "CapitalLettersNotAllowed",
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testController.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_PCIE,
					NvmeControllerId: proto.Int32(1),
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("user-settable ID must only contain lowercase, numbers and hyphens (%v)", "got: 'C' in position 0"),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"valid request with invalid SPDK response": {
			id: testControllerID,
			in: &pb.NvmeController{
				Spec: spec,
			},
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"name": "NvmeEmu0pf0", "cntlid": -1}}`},
			errCode: codes.InvalidArgument,
			errMsg:  fmt.Sprintf("Could not create CTRL: %v", testControllerName),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"valid request with empty SPDK response": {
			id: testControllerID,
			in: &pb.NvmeController{
				Spec: spec,
			},
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_create: %v", "EOF"),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"valid request with ID mismatch SPDK response": {
			id: testControllerID,
			in: &pb.NvmeController{
				Spec: spec,
			},
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":{"name": "NvmeEmu0pf0", "cntlid": 17}}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_create: %v", "json response ID mismatch"),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"valid request with error code from SPDK response": {
			id: testControllerID,
			in: &pb.NvmeController{
				Spec: spec,
			},
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":-32602,"message":"Invalid parameters"}}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_create: %v", "json response error: Invalid parameters"),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"valid request with valid SPDK response": {
			id: testControllerID,
			in: &pb.NvmeController{
				Spec: controllerSpec,
			},
			out: &pb.NvmeController{
				Name: testControllerName,
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testController.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_PCIE,
					NvmeControllerId: proto.Int32(17),
				},
				Status: &pb.NvmeControllerStatus{
					Active: true,
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"name": "NvmeEmu0pf0", "cntlid": 17}}`},
			errCode: codes.OK,
			errMsg:  "",
			exist:   false,
			subsys:  testSubsystemName,
		},
		"already exists": {
			id: testControllerID,
			in: &pb.NvmeController{
				Spec: spec,
			},
			out:     &testControllerWithStatus,
			spdk:    []string{},
			errCode: codes.OK,
			errMsg:  "",
			exist:   true,
			subsys:  testSubsystemName,
		},
		"malformed subsystem name": {
			id: testControllerID,
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint:         testController.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_PCIE,
					NvmeControllerId: proto.Int32(1),
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("segment '%s': not a valid DNS name", "-ABC-DEF"),
			exist:   false,
			subsys:  "-ABC-DEF",
		},
		"no required ctrl field": {
			id:      testControllerID,
			in:      nil,
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: nvme_controller",
			exist:   false,
			subsys:  testSubsystemName,
		},
		"no required parent field": {
			id: testControllerID,
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					NvmeControllerId: proto.Int32(1),
					Endpoint:         testController.Spec.Endpoint,
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_PCIE,
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: parent",
			exist:   false,
			subsys:  "",
		},
		"not supported transport type": {
			id: testControllerID,
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint: &pb.NvmeControllerSpec_FabricsId{
						FabricsId: &pb.FabricsEndpoint{
							Traddr:  "127.0.0.1",
							Trsvcid: "4420",
							Adrfam:  pb.NvmeAddressFamily_NVME_ADRFAM_IPV4,
						},
					},
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_TCP,
					NvmeControllerId: proto.Int32(1),
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("not supported transport type: %v", pb.NvmeTransportType_NVME_TRANSPORT_TCP),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"not corresponding endpoint for pcie transport type": {
			id: testControllerID,
			in: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					Endpoint: &pb.NvmeControllerSpec_FabricsId{
						FabricsId: &pb.FabricsEndpoint{
							Traddr:  "127.0.0.1",
							Trsvcid: "4420",
							Adrfam:  pb.NvmeAddressFamily_NVME_ADRFAM_IPV4,
						},
					},
					Trtype:           pb.NvmeTransportType_NVME_TRANSPORT_PCIE,
					NvmeControllerId: proto.Int32(1),
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "invalid endpoint type passed for transport",
			exist:   false,
			subsys:  testSubsystemName,
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = utils.ProtoClone(&testSubsystemWithStatus)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = utils.ProtoClone(&testNamespaceWithStatus)
			if tt.exist {
				testEnv.opiSpdkServer.Controllers[testControllerName] = utils.ProtoClone(&testControllerWithStatus)
			}
			if tt.out != nil {
				tt.out = utils.ProtoClone(tt.out)
				tt.out.Name = testControllerName
			}

			request := &pb.CreateNvmeControllerRequest{Parent: tt.subsys, NvmeController: tt.in, NvmeControllerId: tt.id}
			response, err := testEnv.client.CreateNvmeController(testEnv.ctx, request)

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

func TestFrontEnd_DeleteNvmeController(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     *emptypb.Empty
		spdk    []string
		errCode codes.Code
		errMsg  string
		missing bool
	}{
		"valid request with invalid SPDK response": {
			in:      testControllerName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			errCode: codes.InvalidArgument,
			errMsg:  fmt.Sprintf("Could not delete NQN:ID %v", "nqn.2022-09.io.spdk:opi3:17"),
			missing: false,
		},
		"valid request with empty SPDK response": {
			in:      testControllerName,
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_delete: %v", "EOF"),
			missing: false,
		},
		"valid request with ID mismatch SPDK response": {
			in:      testControllerName,
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":false}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_delete: %v", "json response ID mismatch"),
			missing: false,
		},
		"valid request with error code from SPDK response": {
			in:      testControllerName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":false}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_delete: %v", "json response error: myopierr"),
			missing: false,
		},
		"valid request with valid SPDK response": {
			in:      testControllerName,
			out:     &emptypb.Empty{},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`}, // `{"jsonrpc": "2.0", "id": 1, "result": True}`,
			errCode: codes.OK,
			errMsg:  "",
			missing: false,
		},
		"valid request with unknown key": {
			in:      "unknown-controller-id",
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("error finding controller %v", "unknown-controller-id"),
			missing: false,
		},
		"unknown key with missing allowed": {
			in:      "unknown-id",
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

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = utils.ProtoClone(&testSubsystemWithStatus)
			testEnv.opiSpdkServer.Controllers[testControllerName] = utils.ProtoClone(&testControllerWithStatus)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = utils.ProtoClone(&testNamespaceWithStatus)

			request := &pb.DeleteNvmeControllerRequest{Name: tt.in, AllowMissing: tt.missing}
			response, err := testEnv.client.DeleteNvmeController(testEnv.ctx, request)

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

func TestFrontEnd_UpdateNvmeController(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		mask    *fieldmaskpb.FieldMask
		in      *pb.NvmeController
		out     *pb.NvmeController
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"invalid fieldmask": {
			mask: &fieldmaskpb.FieldMask{Paths: []string{"*", "author"}},
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: testController.Spec,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("invalid field path: %s", "'*' must not be used with other paths"),
		},
		"unimplemented method": {
			mask: nil,
			in: &pb.NvmeController{
				Name: testControllerName,
				Spec: testController.Spec,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unimplemented,
			errMsg:  fmt.Sprintf("%v method is not implemented", "UpdateNvmeController"),
		},
		"valid request with unknown key": {
			mask: nil,
			in: &pb.NvmeController{
				Name: frontend.ResourceIDToControllerName(testSubsystemID, "unknown-controller-id"),
				Spec: testController.Spec,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find key %v", frontend.ResourceIDToControllerName(testSubsystemID, "unknown-controller-id")),
		},
		"malformed name": {
			mask: nil,
			in: &pb.NvmeController{
				Name: "-ABC-DEF",
				Spec: testController.Spec,
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

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = utils.ProtoClone(&testSubsystemWithStatus)
			testEnv.opiSpdkServer.Controllers[testControllerName] = utils.ProtoClone(&testControllerWithStatus)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = utils.ProtoClone(&testNamespaceWithStatus)

			request := &pb.UpdateNvmeControllerRequest{NvmeController: tt.in, UpdateMask: tt.mask}
			response, err := testEnv.client.UpdateNvmeController(testEnv.ctx, request)

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

func TestFrontEnd_ListNvmeControllers(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     []*pb.NvmeController
		spdk    []string
		errCode codes.Code
		errMsg  string
		size    int32
		token   string
	}{
		"valid request with empty result SPDK response": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    0,
			token:   "",
		},
		"valid request with empty SPDK response": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "EOF"),
			size:    0,
			token:   "",
		},
		"valid request with ID mismatch SPDK response": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "json response ID mismatch"),
			size:    0,
			token:   "",
		},
		"valid request with error code from SPDK response": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "json response error: myopierr"),
			size:    0,
			token:   "",
		},
		"pagination negative": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{},
			errCode: codes.InvalidArgument,
			errMsg:  "negative PageSize is not allowed",
			size:    -10,
			token:   "",
		},
		"pagination error": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find pagination token %s", "unknown-pagination-token"),
			size:    0,
			token:   "unknown-pagination-token",
		},
		"pagination": {
			in: testSubsystemName,
			out: []*pb.NvmeController{
				{
					Spec: &pb.NvmeControllerSpec{
						NvmeControllerId: proto.Int32(1),
					},
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 1, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 1, "pci_bdf": "ca:00.3"},{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 2, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 2, "pci_bdf": "ca:00.4"},{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 3, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 3, "pci_bdf": "ca:00.5"}]}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    1,
			token:   "",
		},
		"pagination overflow": {
			in: testSubsystemName,
			out: []*pb.NvmeController{
				{
					Spec: &pb.NvmeControllerSpec{
						NvmeControllerId: proto.Int32(1),
					},
				},
				{
					Spec: &pb.NvmeControllerSpec{
						NvmeControllerId: proto.Int32(2),
					},
				},
				{
					Spec: &pb.NvmeControllerSpec{
						NvmeControllerId: proto.Int32(3),
					},
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 1, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 1, "pci_bdf": "ca:00.3"},{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 2, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 2, "pci_bdf": "ca:00.4"},{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 3, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 3, "pci_bdf": "ca:00.5"}]}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    1000,
			token:   "",
		},
		"pagination offset": {
			in: testSubsystemName,
			out: []*pb.NvmeController{
				{
					Spec: &pb.NvmeControllerSpec{
						NvmeControllerId: proto.Int32(2),
					},
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 1, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 1, "pci_bdf": "ca:00.3"},{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 2, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 2, "pci_bdf": "ca:00.4"},{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 3, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 3, "pci_bdf": "ca:00.5"}]}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    1,
			token:   "existing-pagination-token",
		},
		"valid request with valid SPDK response": {
			in: testSubsystemName,
			out: []*pb.NvmeController{
				{
					Spec: &pb.NvmeControllerSpec{
						NvmeControllerId: proto.Int32(1),
					},
				},
				{
					Spec: &pb.NvmeControllerSpec{
						NvmeControllerId: proto.Int32(2),
					},
				},
				{
					Spec: &pb.NvmeControllerSpec{
						NvmeControllerId: proto.Int32(3),
					},
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 1, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 1, "pci_bdf": "ca:00.3"},{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 2, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 2, "pci_bdf": "ca:00.4"},{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 3, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 3, "pci_bdf": "ca:00.5"}]}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    0,
			token:   "",
		},
		"valid request with unknown key": {
			in:      frontend.ResourceIDToControllerName(testSubsystemID, "unknown-controller-id"),
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find key %v", frontend.ResourceIDToControllerName(testSubsystemID, "unknown-controller-id")),
			size:    0,
			token:   "",
		},
		"no required field": {
			in:      "",
			out:     []*pb.NvmeController{},
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

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = utils.ProtoClone(&testSubsystemWithStatus)
			testEnv.opiSpdkServer.Controllers[testControllerName] = utils.ProtoClone(&testControllerWithStatus)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = utils.ProtoClone(&testNamespaceWithStatus)
			testEnv.opiSpdkServer.Pagination["existing-pagination-token"] = 1

			request := &pb.ListNvmeControllersRequest{Parent: tt.in, PageSize: tt.size, PageToken: tt.token}
			response, err := testEnv.client.ListNvmeControllers(testEnv.ctx, request)

			if !utils.EqualProtoSlices(response.GetNvmeControllers(), tt.out) {
				t.Error("response: expected", tt.out, "received", response.GetNvmeControllers())
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

func TestFrontEnd_GetNvmeController(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     *pb.NvmeController
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"valid request with invalid SPDK response": {
			in:      testControllerName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.InvalidArgument,
			errMsg:  fmt.Sprintf("Could not find NvmeControllerId: %v", "17"),
		},
		"valid request with empty SPDK response": {
			in:      testControllerName,
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "EOF"),
		},
		"valid request with ID mismatch SPDK response": {
			in:      testControllerName,
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "json response ID mismatch"),
		},
		"valid request with error code from SPDK response": {
			in:      testControllerName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_list: %v", "json response error: myopierr"),
		},
		"valid request with valid SPDK response": {
			in: testControllerName,
			out: &pb.NvmeController{
				Spec: &pb.NvmeControllerSpec{
					NvmeControllerId: proto.Int32(17),
				},
				Status: &pb.NvmeControllerStatus{
					Active: true,
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 1, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 1, "pci_bdf": "ca:00.3"},{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 17, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 2, "pci_bdf": "ca:00.4"},{"subnqn": "nqn.2022-09.io.spdk:opi3", "cntlid": 3, "name": "NvmeEmu0pf1", "type": "nvme", "pci_index": 3, "pci_bdf": "ca:00.5"}]}`},
			errCode: codes.OK,
			errMsg:  "",
		},
		"valid request with unknown key": {
			in:      frontend.ResourceIDToControllerName(testSubsystemID, "unknown-controller-id"),
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find key %v", frontend.ResourceIDToControllerName(testSubsystemID, "unknown-controller-id")),
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

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = utils.ProtoClone(&testSubsystemWithStatus)
			testEnv.opiSpdkServer.Controllers[testControllerName] = utils.ProtoClone(&testControllerWithStatus)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = utils.ProtoClone(&testNamespaceWithStatus)

			request := &pb.GetNvmeControllerRequest{Name: tt.in}
			response, err := testEnv.client.GetNvmeController(testEnv.ctx, request)

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

func TestFrontEnd_StatsNvmeController(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     *pb.StatsNvmeControllerResponse
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"unimplemented method": {
			in:      testControllerName,
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unimplemented,
			errMsg:  fmt.Sprintf("%v method is not implemented", "StatsNvmeController"),
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

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = utils.ProtoClone(&testSubsystemWithStatus)
			testEnv.opiSpdkServer.Controllers[testControllerName] = utils.ProtoClone(&testControllerWithStatus)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = utils.ProtoClone(&testNamespaceWithStatus)

			request := &pb.StatsNvmeControllerRequest{Name: tt.in}
			response, err := testEnv.client.StatsNvmeController(testEnv.ctx, request)

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
