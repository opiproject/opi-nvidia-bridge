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
	"github.com/opiproject/opi-spdk-bridge/pkg/server"
)

func TestFrontEnd_CreateNvmeSubsystem(t *testing.T) {
	spec := &pb.NvmeSubsystemSpec{
		Nqn:          "nqn.2022-09.io.spdk:opi3",
		SerialNumber: "OpiSerialNumber",
		ModelNumber:  "OpiModelNumber",
	}
	tests := map[string]struct {
		id      string
		in      *pb.NvmeSubsystem
		out     *pb.NvmeSubsystem
		spdk    []string
		errCode codes.Code
		errMsg  string
		exist   bool
	}{
		"illegal resource_id": {
			"CapitalLettersNotAllowed",
			&pb.NvmeSubsystem{
				Spec: spec,
			},
			nil,
			[]string{},
			codes.Unknown,
			fmt.Sprintf("user-settable ID must only contain lowercase, numbers and hyphens (%v)", "got: 'C' in position 0"),
			false,
		},
		"valid request with invalid SPDK response": {
			testSubsystemID,
			&pb.NvmeSubsystem{
				Spec: spec,
			},
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			codes.InvalidArgument,
			fmt.Sprintf("Could not create NQN: %v", "nqn.2022-09.io.spdk:opi3"),
			false,
		},
		"valid request with empty SPDK response": {
			testSubsystemID,
			&pb.NvmeSubsystem{
				Spec: spec,
			},
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_create: %v", "EOF"),
			false,
		},
		"valid request with ID mismatch SPDK response": {
			testSubsystemID,
			&pb.NvmeSubsystem{
				Spec: spec,
			},
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":false}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_create: %v", "json response ID mismatch"),
			false,
		},
		"valid request with error code from SPDK response": {
			testSubsystemID,
			&pb.NvmeSubsystem{
				Spec: spec,
			},
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":false}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_create: %v", "json response error: myopierr"),
			false,
		},
		"valid request with error code from SPDK version response": {
			testSubsystemID,
			&pb.NvmeSubsystem{
				Spec: spec,
			},
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`, `{"id":%d,"error":{"code":1,"message":"myopierr"},"result":false}`},
			codes.Unknown,
			fmt.Sprintf("spdk_get_version: %v", "json response error: myopierr"),
			false,
		},
		"valid request with valid SPDK response": {
			testSubsystemID,
			&pb.NvmeSubsystem{
				Spec: spec,
			},
			&pb.NvmeSubsystem{
				Name: testSubsystemName,
				Spec: &pb.NvmeSubsystemSpec{
					Nqn:          "nqn.2022-09.io.spdk:opi3",
					SerialNumber: "OpiSerialNumber",
					ModelNumber:  "OpiModelNumber",
				},
				Status: &pb.NvmeSubsystemStatus{
					FirmwareRevision: "SPDK v20.10",
				},
			},
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`,
				`{"jsonrpc":"2.0","id":%d,"result":{"version":"SPDK v20.10","fields":{"major":20,"minor":10,"patch":0,"suffix":""}}}`},
			codes.OK,
			"",
			false,
		},
		"already exists": {
			testSubsystemID,
			&pb.NvmeSubsystem{
				Spec: spec,
			},
			&testSubsystem,
			[]string{},
			codes.OK,
			"",
			true,
		},
		"no required field": {
			testControllerID,
			nil,
			nil,
			[]string{},
			codes.Unknown,
			"missing required field: nvme_subsystem",
			false,
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.Controllers[testControllerName] = &testController
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = &testNamespace
			if tt.exist {
				testEnv.opiSpdkServer.Subsystems[testSubsystemName] = &testSubsystem
				testEnv.opiSpdkServer.Subsystems[testSubsystemName].Name = testSubsystemName
			}
			if tt.out != nil {
				tt.out.Name = testSubsystemName
			}

			request := &pb.CreateNvmeSubsystemRequest{NvmeSubsystem: tt.in, NvmeSubsystemId: tt.id}
			response, err := testEnv.client.CreateNvmeSubsystem(testEnv.ctx, request)

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

func TestFrontEnd_DeleteNvmeSubsystem(t *testing.T) {
	tests := map[string]struct {
		in      string
		out     *emptypb.Empty
		spdk    []string
		errCode codes.Code
		errMsg  string
		missing bool
	}{
		"valid request with invalid SPDK response": {
			testSubsystemName,
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			codes.InvalidArgument,
			fmt.Sprintf("Could not delete NQN: %v", "nqn.2022-09.io.spdk:opi3"),
			false,
		},
		"valid request with empty SPDK response": {
			testSubsystemName,
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_delete: %v", "EOF"),
			false,
		},
		"valid request with ID mismatch SPDK response": {
			testSubsystemName,
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":false}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_delete: %v", "json response ID mismatch"),
			false,
		},
		"valid request with error code from SPDK response": {
			testSubsystemName,
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":false}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_delete: %v", "json response error: myopierr"),
			false,
		},
		"valid request with valid SPDK response": {
			testSubsystemName,
			&emptypb.Empty{},
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`}, // `{"jsonrpc": "2.0", "id": 1, "result": True}`,
			codes.OK,
			"",
			false,
		},
		"valid request with unknown key": {
			"unknown-subsystem-id",
			nil,
			[]string{},
			codes.NotFound,
			fmt.Sprintf("unable to find key %v", "unknown-subsystem-id"),
			false,
		},
		"unknown key with missing allowed": {
			"unknown-id",
			&emptypb.Empty{},
			[]string{},
			codes.OK,
			"",
			true,
		},
		"malformed name": {
			"-ABC-DEF",
			&emptypb.Empty{},
			[]string{},
			codes.Unknown,
			fmt.Sprintf("segment '%s': not a valid DNS name", "-ABC-DEF"),
			false,
		},
		"no required field": {
			"",
			nil,
			[]string{},
			codes.Unknown,
			"missing required field: name",
			false,
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = &testSubsystem
			testEnv.opiSpdkServer.Controllers[testControllerName] = &testController
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = &testNamespace

			request := &pb.DeleteNvmeSubsystemRequest{Name: tt.in, AllowMissing: tt.missing}
			response, err := testEnv.client.DeleteNvmeSubsystem(testEnv.ctx, request)

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

func TestFrontEnd_UpdateNvmeSubsystem(t *testing.T) {
	tests := map[string]struct {
		mask    *fieldmaskpb.FieldMask
		in      *pb.NvmeSubsystem
		out     *pb.NvmeSubsystem
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"invalid fieldmask": {
			&fieldmaskpb.FieldMask{Paths: []string{"*", "author"}},
			&pb.NvmeSubsystem{
				Name: testSubsystemName,
			},
			nil,
			[]string{},
			codes.Unknown,
			fmt.Sprintf("invalid field path: %s", "'*' must not be used with other paths"),
		},
		"unimplemented method": {
			nil,
			&pb.NvmeSubsystem{
				Name: testSubsystemName,
			},
			nil,
			[]string{},
			codes.Unimplemented,
			fmt.Sprintf("%v method is not implemented", "UpdateNvmeSubsystem"),
		},
		"valid request with unknown key": {
			nil,
			&pb.NvmeSubsystem{
				Name: server.ResourceIDToVolumeName("unknown-id"),
				Spec: &pb.NvmeSubsystemSpec{
					Nqn: "nqn.2022-09.io.spdk:opi3",
				},
			},
			nil,
			[]string{},
			codes.NotFound,
			fmt.Sprintf("unable to find key %v", server.ResourceIDToVolumeName("unknown-id")),
		},
		"malformed name": {
			nil,
			&pb.NvmeSubsystem{Name: "-ABC-DEF"},
			nil,
			[]string{},
			codes.Unknown,
			fmt.Sprintf("segment '%s': not a valid DNS name", "-ABC-DEF"),
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = &testSubsystem
			testEnv.opiSpdkServer.Controllers[testControllerName] = &testController
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = &testNamespace

			request := &pb.UpdateNvmeSubsystemRequest{NvmeSubsystem: tt.in, UpdateMask: tt.mask}
			response, err := testEnv.client.UpdateNvmeSubsystem(testEnv.ctx, request)

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

func TestFrontEnd_ListNvmeSubsystem(t *testing.T) {
	testParent := "todo"
	tests := map[string]struct {
		in      string
		out     []*pb.NvmeSubsystem
		spdk    []string
		errCode codes.Code
		errMsg  string
		size    int32
		token   string
	}{
		"valid request with empty result SPDK response": {
			testParent,
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			codes.OK,
			"",
			0,
			"",
		},
		"valid request with empty SPDK response": {
			testParent,
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "EOF"),
			0,
			"",
		},
		"valid request with ID mismatch SPDK response": {
			testParent,
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "json response ID mismatch"),
			0,
			"",
		},
		"valid request with error code from SPDK response": {
			testParent,
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "json response error: myopierr"),
			0,
			"",
		},
		"pagination negative": {
			testParent,
			nil,
			[]string{},
			codes.InvalidArgument,
			"negative PageSize is not allowed",
			-10,
			"",
		},
		"pagination error": {
			testParent,
			nil,
			[]string{},
			codes.NotFound,
			fmt.Sprintf("unable to find pagination token %s", "unknown-pagination-token"),
			0,
			"unknown-pagination-token",
		},
		"pagination": {
			testParent,
			[]*pb.NvmeSubsystem{
				{
					Spec: &pb.NvmeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi1",
						SerialNumber: "OpiSerialNumber1",
						ModelNumber:  "OpiModelNumber1",
					},
				},
			},
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"nqn": "nqn.2022-09.io.spdk:opi1", "serial_number": "OpiSerialNumber1", "model_number": "OpiModelNumber1"},{"nqn": "nqn.2022-09.io.spdk:opi2", "serial_number": "OpiSerialNumber2", "model_number": "OpiModelNumber2"},{"nqn": "nqn.2022-09.io.spdk:opi3", "serial_number": "OpiSerialNumber3", "model_number": "OpiModelNumber3"}]}`},
			codes.OK,
			"",
			1,
			"",
		},
		"pagination overflow": {
			testParent,
			[]*pb.NvmeSubsystem{
				{
					Spec: &pb.NvmeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi1",
						SerialNumber: "OpiSerialNumber1",
						ModelNumber:  "OpiModelNumber1",
					},
				},
				{
					Spec: &pb.NvmeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi2",
						SerialNumber: "OpiSerialNumber2",
						ModelNumber:  "OpiModelNumber2",
					},
				},
				{
					Spec: &pb.NvmeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi3",
						SerialNumber: "OpiSerialNumber3",
						ModelNumber:  "OpiModelNumber3",
					},
				},
			},
			// {'jsonrpc': '2.0', 'id': 1, 'result': [{'nqn': 'nqn.2020-12.mlnx.snap', 'serial_number': 'Mellanox_Nvme_SNAP', 'model_number': 'Mellanox Nvme SNAP Controller', 'controllers': [{'name': 'NvmeEmu0pf1', 'cntlid': 0, 'pci_bdf': 'ca:00.3', 'pci_index': 1}]}]}
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"nqn": "nqn.2022-09.io.spdk:opi1", "serial_number": "OpiSerialNumber1", "model_number": "OpiModelNumber1"},{"nqn": "nqn.2022-09.io.spdk:opi2", "serial_number": "OpiSerialNumber2", "model_number": "OpiModelNumber2"},{"nqn": "nqn.2022-09.io.spdk:opi3", "serial_number": "OpiSerialNumber3", "model_number": "OpiModelNumber3"}]}`},
			codes.OK,
			"",
			1000,
			"",
		},
		"pagination offset": {
			testParent,
			[]*pb.NvmeSubsystem{
				{
					Spec: &pb.NvmeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi2",
						SerialNumber: "OpiSerialNumber2",
						ModelNumber:  "OpiModelNumber2",
					},
				},
			},
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"nqn": "nqn.2022-09.io.spdk:opi1", "serial_number": "OpiSerialNumber1", "model_number": "OpiModelNumber1"},{"nqn": "nqn.2022-09.io.spdk:opi2", "serial_number": "OpiSerialNumber2", "model_number": "OpiModelNumber2"},{"nqn": "nqn.2022-09.io.spdk:opi3", "serial_number": "OpiSerialNumber3", "model_number": "OpiModelNumber3"}]}`},
			codes.OK,
			"",
			1,
			"existing-pagination-token",
		},
		"valid request with valid SPDK response": {
			testParent,
			[]*pb.NvmeSubsystem{
				{
					Spec: &pb.NvmeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi1",
						SerialNumber: "OpiSerialNumber1",
						ModelNumber:  "OpiModelNumber1",
					},
				},
				{
					Spec: &pb.NvmeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi2",
						SerialNumber: "OpiSerialNumber2",
						ModelNumber:  "OpiModelNumber2",
					},
				},
				{
					Spec: &pb.NvmeSubsystemSpec{
						Nqn:          "nqn.2022-09.io.spdk:opi3",
						SerialNumber: "OpiSerialNumber3",
						ModelNumber:  "OpiModelNumber3",
					},
				},
			},
			// {'jsonrpc': '2.0', 'id': 1, 'result': [{'nqn': 'nqn.2020-12.mlnx.snap', 'serial_number': 'Mellanox_Nvme_SNAP', 'model_number': 'Mellanox Nvme SNAP Controller', 'controllers': [{'name': 'NvmeEmu0pf1', 'cntlid': 0, 'pci_bdf': 'ca:00.3', 'pci_index': 1}]}]}
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"nqn": "nqn.2022-09.io.spdk:opi1", "serial_number": "OpiSerialNumber1", "model_number": "OpiModelNumber1"},{"nqn": "nqn.2022-09.io.spdk:opi2", "serial_number": "OpiSerialNumber2", "model_number": "OpiModelNumber2"},{"nqn": "nqn.2022-09.io.spdk:opi3", "serial_number": "OpiSerialNumber3", "model_number": "OpiModelNumber3"}]}`},
			codes.OK,
			"",
			0,
			"",
		},
		"no required field": {
			"",
			[]*pb.NvmeSubsystem{},
			[]string{},
			codes.Unknown,
			"missing required field: parent",
			0,
			"",
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = &testSubsystem
			testEnv.opiSpdkServer.Controllers[testControllerName] = &testController
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = &testNamespace
			testEnv.opiSpdkServer.Pagination["existing-pagination-token"] = 1

			request := &pb.ListNvmeSubsystemsRequest{Parent: tt.in, PageSize: tt.size, PageToken: tt.token}
			response, err := testEnv.client.ListNvmeSubsystems(testEnv.ctx, request)

			if !server.EqualProtoSlices(response.GetNvmeSubsystems(), tt.out) {
				t.Error("response: expected", tt.out, "received", response.GetNvmeSubsystems())
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

func TestFrontEnd_GetNvmeSubsystem(t *testing.T) {
	tests := map[string]struct {
		in      string
		out     *pb.NvmeSubsystem
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"valid request with invalid SPDK response": {
			testSubsystemName,
			nil,
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			codes.InvalidArgument,
			fmt.Sprintf("Could not find NQN: %v", "nqn.2022-09.io.spdk:opi3"),
		},
		"valid request with empty SPDK response": {
			testSubsystemName,
			nil,
			[]string{""},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "EOF"),
		},
		"valid request with ID mismatch SPDK response": {
			testSubsystemName,
			nil,
			[]string{`{"id":0,"error":{"code":0,"message":""},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "json response ID mismatch"),
		},
		"valid request with error code from SPDK response": {
			testSubsystemName,
			nil,
			[]string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":[]}`},
			codes.Unknown,
			fmt.Sprintf("subsystem_nvme_list: %v", "json response error: myopierr"),
		},
		"valid request with valid SPDK response": {
			testSubsystemName,
			&pb.NvmeSubsystem{
				Spec: &pb.NvmeSubsystemSpec{
					Nqn:          "nqn.2022-09.io.spdk:opi3",
					SerialNumber: "OpiSerialNumber3",
					ModelNumber:  "OpiModelNumber3",
				},
				Status: &pb.NvmeSubsystemStatus{
					FirmwareRevision: "TBD",
				},
			},
			// {'jsonrpc': '2.0', 'id': 1, 'result': [{'nqn': 'nqn.2020-12.mlnx.snap', 'serial_number': 'Mellanox_Nvme_SNAP', 'model_number': 'Mellanox Nvme SNAP Controller', 'controllers': [{'name': 'NvmeEmu0pf1', 'cntlid': 0, 'pci_bdf': 'ca:00.3', 'pci_index': 1}]}]}
			[]string{`{"id":%d,"error":{"code":0,"message":""},"result":[{"nqn": "nqn.2022-09.io.spdk:opi1", "serial_number": "OpiSerialNumber1", "model_number": "OpiModelNumber1"},{"nqn": "nqn.2022-09.io.spdk:opi2", "serial_number": "OpiSerialNumber2", "model_number": "OpiModelNumber2"},{"nqn": "nqn.2022-09.io.spdk:opi3", "serial_number": "OpiSerialNumber3", "model_number": "OpiModelNumber3"}]}`},
			codes.OK,
			"",
		},
		"valid request with unknown key": {
			server.ResourceIDToVolumeName("unknown-subsystem-id"),
			nil,
			[]string{},
			codes.NotFound,
			fmt.Sprintf("unable to find key %v", server.ResourceIDToVolumeName("unknown-subsystem-id")),
		},
		"malformed name": {
			"-ABC-DEF",
			nil,
			[]string{},
			codes.Unknown,
			fmt.Sprintf("segment '%s': not a valid DNS name", "-ABC-DEF"),
		},
		"no required field": {
			"",
			nil,
			[]string{},
			codes.Unknown,
			"missing required field: name",
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = &testSubsystem
			testEnv.opiSpdkServer.Controllers[testControllerName] = &testController
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = &testNamespace

			request := &pb.GetNvmeSubsystemRequest{Name: tt.in}
			response, err := testEnv.client.GetNvmeSubsystem(testEnv.ctx, request)

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

func TestFrontEnd_NvmeSubsystemStats(t *testing.T) {
	tests := map[string]struct {
		in      string
		out     *pb.NvmeSubsystemStatsResponse
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"unimplemented method": {
			testSubsystemName,
			nil,
			[]string{},
			codes.Unimplemented,
			fmt.Sprintf("%v method is not implemented", "UpdateNvmeSubsystem"),
		},
		"malformed name": {
			"-ABC-DEF",
			nil,
			[]string{},
			codes.Unknown,
			fmt.Sprintf("segment '%s': not a valid DNS name", "-ABC-DEF"),
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = &testSubsystem
			testEnv.opiSpdkServer.Controllers[testControllerName] = &testController
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = &testNamespace

			request := &pb.NvmeSubsystemStatsRequest{SubsystemNameRef: tt.in}
			response, err := testEnv.client.NvmeSubsystemStats(testEnv.ctx, request)

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
