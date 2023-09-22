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

	pc "github.com/opiproject/opi-api/common/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/frontend"
	"github.com/opiproject/opi-spdk-bridge/pkg/server"
)

func TestFrontEnd_CreateNvmeNamespace(t *testing.T) {
	spec := &pb.NvmeNamespaceSpec{
		HostNsid:      0,
		VolumeNameRef: "Malloc1",
		Uuid:          &pc.Uuid{Value: "1b4e28ba-2fa1-11d2-883f-b9a761bde3fb"},
		Nguid:         "1b4e28ba-2fa1-11d2-883f-b9a761bde3fb",
		Eui64:         1967554867335598546,
	}
	namespaceSpec := &pb.NvmeNamespaceSpec{
		HostNsid:      22,
		VolumeNameRef: "Malloc1",
		Uuid:          &pc.Uuid{Value: "1b4e28ba-2fa1-11d2-883f-b9a761bde3fb"},
		Nguid:         "1b4e28ba-2fa1-11d2-883f-b9a761bde3fb",
		Eui64:         1967554867335598546,
	}
	t.Cleanup(server.CheckTestProtoObjectsNotChanged(spec, namespaceSpec)(t, t.Name()))
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))

	tests := map[string]struct {
		id      string
		in      *pb.NvmeNamespace
		out     *pb.NvmeNamespace
		spdk    []string
		errCode codes.Code
		errMsg  string
		exist   bool
		subsys  string
	}{
		"illegal resource_id": {
			id: "CapitalLettersNotAllowed",
			in: &pb.NvmeNamespace{
				Spec: spec,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("user-settable ID must only contain lowercase, numbers and hyphens (%v)", "got: 'C' in position 0"),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"valid request with invalid SPDK response": {
			id: testNamespaceID,
			in: &pb.NvmeNamespace{
				Spec: spec,
			},
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			errCode: codes.InvalidArgument,
			errMsg:  fmt.Sprintf("Could not create NS: %v", testNamespaceName),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"valid request with empty SPDK response": {
			id: testNamespaceID,
			in: &pb.NvmeNamespace{
				Spec: spec,
			},
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_attach: %v", "EOF"),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"valid request with ID mismatch SPDK response": {
			id: testNamespaceID,
			in: &pb.NvmeNamespace{
				Spec: spec,
			},
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":false}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_attach: %v", "json response ID mismatch"),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"valid request with error code from SPDK response": {
			id: testNamespaceID,
			in: &pb.NvmeNamespace{
				Spec: spec,
			},
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":false}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_attach: %v", "json response error: myopierr"),
			exist:   false,
			subsys:  testSubsystemName,
		},
		"valid request with valid SPDK response": {
			id: testNamespaceID,
			in: &pb.NvmeNamespace{
				Spec: namespaceSpec,
			},
			out: &pb.NvmeNamespace{
				Name: testNamespaceName,
				Spec: &pb.NvmeNamespaceSpec{
					HostNsid:      22,
					VolumeNameRef: "Malloc1",
					Uuid:          &pc.Uuid{Value: "1b4e28ba-2fa1-11d2-883f-b9a761bde3fb"},
					Nguid:         "1b4e28ba-2fa1-11d2-883f-b9a761bde3fb",
					Eui64:         1967554867335598546,
				},
				Status: &pb.NvmeNamespaceStatus{
					PciState:     2,
					PciOperState: 1,
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`},
			errCode: codes.OK,
			errMsg:  "",
			exist:   false,
			subsys:  testSubsystemName,
		},
		"already exists": {
			id: testNamespaceID,
			in: &pb.NvmeNamespace{
				Spec: spec,
			},
			out:     &testNamespace,
			spdk:    []string{},
			errCode: codes.OK,
			errMsg:  "",
			exist:   true,
			subsys:  testSubsystemName,
		},
		"malformed subsystem name": {
			id: testNamespaceID,
			in: &pb.NvmeNamespace{
				Spec: &pb.NvmeNamespaceSpec{
					VolumeNameRef: "TBD",
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("segment '%s': not a valid DNS name", "-ABC-DEF"),
			exist:   false,
			subsys:  "-ABC-DEF",
		},
		"no required ns field": {
			id:      testNamespaceID,
			in:      nil,
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: nvme_namespace",
			exist:   false,
			subsys:  testSubsystemName,
		},
		"no required parent field": {
			id: testNamespaceID,
			in: &pb.NvmeNamespace{
				Spec: &pb.NvmeNamespaceSpec{
					VolumeNameRef: "TBD",
				},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: parent",
			exist:   false,
			subsys:  "",
		},
		"no required volume field": {
			id: testNamespaceID,
			in: &pb.NvmeNamespace{
				Spec: &pb.NvmeNamespaceSpec{},
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  "missing required field: nvme_namespace.spec.volume_name_ref",
			exist:   false,
			subsys:  testSubsystemName,
		},
	}

	// run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testEnv := createTestEnvironment(tt.spdk)
			defer testEnv.Close()

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = server.ProtoClone(&testSubsystem)
			testEnv.opiSpdkServer.Subsystems[testSubsystemName].Name = testSubsystemName
			testEnv.opiSpdkServer.Controllers[testControllerName] = server.ProtoClone(&testController)
			testEnv.opiSpdkServer.Controllers[testControllerName].Name = testControllerName
			if tt.exist {
				testEnv.opiSpdkServer.Namespaces[testNamespaceName] = server.ProtoClone(&testNamespace)
				testEnv.opiSpdkServer.Namespaces[testNamespaceName].Name = testNamespaceName
			}
			if tt.out != nil {
				tt.out = server.ProtoClone(tt.out)
				tt.out.Name = testNamespaceName
			}

			request := &pb.CreateNvmeNamespaceRequest{Parent: tt.subsys, NvmeNamespace: tt.in, NvmeNamespaceId: tt.id}
			response, err := testEnv.client.CreateNvmeNamespace(testEnv.ctx, request)

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

func TestFrontEnd_DeleteNvmeNamespace(t *testing.T) {
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
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":false}`},
			errCode: codes.InvalidArgument,
			errMsg:  fmt.Sprintf("Could not delete NS: %v", testNamespaceName),
			missing: false,
		},
		"valid request with empty SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_detach: %v", "EOF"),
			missing: false,
		},
		"valid request with ID mismatch SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":false}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_detach: %v", "json response ID mismatch"),
			missing: false,
		},
		"valid request with error code from SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"},"result":false}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_detach: %v", "json response error: myopierr"),
			missing: false,
		},
		"valid request with valid SPDK response": {
			in:      testNamespaceName,
			out:     &emptypb.Empty{},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":true}`}, // `{"jsonrpc": "2.0", "id": 1, "result": True}`,
			errCode: codes.OK,
			errMsg:  "",
			missing: false,
		},
		"valid request with unknown key": {
			in:      "unknown-namespace-id",
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find key %v", "unknown-namespace-id"),
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

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = server.ProtoClone(&testSubsystem)
			testEnv.opiSpdkServer.Subsystems[testSubsystemName].Name = testSubsystemName
			testEnv.opiSpdkServer.Controllers[testControllerName] = server.ProtoClone(&testController)
			testEnv.opiSpdkServer.Controllers[testControllerName].Name = testControllerName
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = server.ProtoClone(&testNamespace)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName].Name = testNamespaceName

			request := &pb.DeleteNvmeNamespaceRequest{Name: tt.in, AllowMissing: tt.missing}
			response, err := testEnv.client.DeleteNvmeNamespace(testEnv.ctx, request)

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

func TestFrontEnd_UpdateNvmeNamespace(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		mask    *fieldmaskpb.FieldMask
		in      *pb.NvmeNamespace
		out     *pb.NvmeNamespace
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"invalid fieldmask": {
			mask: &fieldmaskpb.FieldMask{Paths: []string{"*", "author"}},
			in: &pb.NvmeNamespace{
				Name: testNamespaceName,
				Spec: testNamespace.Spec,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("invalid field path: %s", "'*' must not be used with other paths"),
		},
		"unimplemented method": {
			mask: nil,
			in: &pb.NvmeNamespace{
				Name: testNamespaceName,
				Spec: testNamespace.Spec,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.Unimplemented,
			errMsg:  fmt.Sprintf("%v method is not implemented", "UpdateNvmeNamespace"),
		},
		"valid request with unknown key": {
			mask: nil,
			in: &pb.NvmeNamespace{
				Name: frontend.ResourceIDToNamespaceName(testSubsystemID, "unknown-namespace-id"),
				Spec: testNamespace.Spec,
			},
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find key %v", frontend.ResourceIDToNamespaceName(testSubsystemID, "unknown-namespace-id")),
		},
		"malformed name": {
			mask: nil,
			in: &pb.NvmeNamespace{
				Name: "-ABC-DEF",
				Spec: testNamespace.Spec,
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

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = server.ProtoClone(&testSubsystem)
			testEnv.opiSpdkServer.Subsystems[testSubsystemName].Name = testSubsystemName
			testEnv.opiSpdkServer.Controllers[testControllerName] = server.ProtoClone(&testController)
			testEnv.opiSpdkServer.Controllers[testControllerName].Name = testControllerName
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = server.ProtoClone(&testNamespace)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName].Name = testNamespaceName

			request := &pb.UpdateNvmeNamespaceRequest{NvmeNamespace: tt.in, UpdateMask: tt.mask}
			response, err := testEnv.client.UpdateNvmeNamespace(testEnv.ctx, request)

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

func TestFrontEnd_ListNvmeNamespaces(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     []*pb.NvmeNamespace
		spdk    []string
		errCode codes.Code
		errMsg  string
		size    int32
		token   string
	}{
		"valid request with empty result SPDK response": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"name":"","cntlid":0,"Namespaces":null}}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    0,
			token:   "",
		},
		"valid request with invalid marshal SPDK response": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_list: %v", "json: cannot unmarshal array into Go value of type models.NvdaControllerNvmeNamespaceListResult"),
			size:    0,
			token:   "",
		},
		"valid request with empty SPDK response": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_list: %v", "EOF"),
			size:    0,
			token:   "",
		},
		"valid request with ID mismatch SPDK response": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":{"name":"","cntlid":0,"Namespaces":null}}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_list: %v", "json response ID mismatch"),
			size:    0,
			token:   "",
		},
		"valid request with error code from SPDK response": {
			in:      testSubsystemName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"}}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_list: %v", "json response error: myopierr"),
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
			out: []*pb.NvmeNamespace{
				{
					Spec: &pb.NvmeNamespaceSpec{
						HostNsid: 11,
					},
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"name": "NvmeEmu0pf1", "cntlid": 1, "Namespaces": [{"nsid": 11, "bdev": "Malloc0", "bdev_type": "spdk", "qn": "", "protocol": ""},{"nsid": 12, "bdev": "Malloc1", "bdev_type": "spdk", "qn": "", "protocol": ""},{"nsid": 13, "bdev": "Malloc2", "bdev_type": "spdk", "qn": "", "protocol": ""}]}}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    1,
			token:   "",
		},
		"pagination overflow": {
			in: testSubsystemName,
			out: []*pb.NvmeNamespace{
				{
					Spec: &pb.NvmeNamespaceSpec{
						HostNsid: 11,
					},
				},
				{
					Spec: &pb.NvmeNamespaceSpec{
						HostNsid: 12,
					},
				},
				{
					Spec: &pb.NvmeNamespaceSpec{
						HostNsid: 13,
					},
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"name": "NvmeEmu0pf1", "cntlid": 1, "Namespaces": [{"nsid": 11, "bdev": "Malloc0", "bdev_type": "spdk", "qn": "", "protocol": ""},{"nsid": 12, "bdev": "Malloc1", "bdev_type": "spdk", "qn": "", "protocol": ""},{"nsid": 13, "bdev": "Malloc2", "bdev_type": "spdk", "qn": "", "protocol": ""}]}}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    1000,
			token:   "",
		},
		"pagination offset": {
			in: testSubsystemName,
			out: []*pb.NvmeNamespace{
				{
					Spec: &pb.NvmeNamespaceSpec{
						HostNsid: 12,
					},
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"name": "NvmeEmu0pf1", "cntlid": 1, "Namespaces": [{"nsid": 11, "bdev": "Malloc0", "bdev_type": "spdk", "qn": "", "protocol": ""},{"nsid": 12, "bdev": "Malloc1", "bdev_type": "spdk", "qn": "", "protocol": ""},{"nsid": 13, "bdev": "Malloc2", "bdev_type": "spdk", "qn": "", "protocol": ""}]}}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    1,
			token:   "existing-pagination-token",
		},
		"valid request with valid SPDK response": {
			in: testSubsystemName,
			out: []*pb.NvmeNamespace{
				{
					Spec: &pb.NvmeNamespaceSpec{
						HostNsid: 11,
					},
				},
				{
					Spec: &pb.NvmeNamespaceSpec{
						HostNsid: 12,
					},
				},
				{
					Spec: &pb.NvmeNamespaceSpec{
						HostNsid: 13,
					},
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"name": "NvmeEmu0pf1", "cntlid": 1, "Namespaces": [{"nsid": 11, "bdev": "Malloc0", "bdev_type": "spdk", "qn": "", "protocol": ""},{"nsid": 12, "bdev": "Malloc1", "bdev_type": "spdk", "qn": "", "protocol": ""},{"nsid": 13, "bdev": "Malloc2", "bdev_type": "spdk", "qn": "", "protocol": ""}]}}`},
			errCode: codes.OK,
			errMsg:  "",
			size:    0,
			token:   "",
		},
		"valid request with unknown key": {
			in:      frontend.ResourceIDToNamespaceName(testSubsystemID, "unknown-namespace-id"),
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find key %v", frontend.ResourceIDToNamespaceName(testSubsystemID, "unknown-namespace-id")),
			size:    0,
			token:   "",
		},
		"no required field": {
			in:      "",
			out:     []*pb.NvmeNamespace{},
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

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = server.ProtoClone(&testSubsystem)
			testEnv.opiSpdkServer.Subsystems[testSubsystemName].Name = testSubsystemName
			testEnv.opiSpdkServer.Controllers[testControllerName] = server.ProtoClone(&testController)
			testEnv.opiSpdkServer.Controllers[testControllerName].Name = testControllerName
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = server.ProtoClone(&testNamespace)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName].Name = testNamespaceName
			testEnv.opiSpdkServer.Pagination["existing-pagination-token"] = 1

			request := &pb.ListNvmeNamespacesRequest{Parent: tt.in, PageSize: tt.size, PageToken: tt.token}
			response, err := testEnv.client.ListNvmeNamespaces(testEnv.ctx, request)

			if !server.EqualProtoSlices(response.GetNvmeNamespaces(), tt.out) {
				t.Error("response: expected", tt.out, "received", response.GetNvmeNamespaces())
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

func TestFrontEnd_GetNvmeNamespace(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     *pb.NvmeNamespace
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"valid request with invalid SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"name":"","cntlid":17,"Namespaces":null}}`},
			errCode: codes.InvalidArgument,
			errMsg:  fmt.Sprintf("Could not find HostNsid: %v", "22"),
		},
		"valid request with invalid marshal SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_list: %v", "json: cannot unmarshal array into Go value of type models.NvdaControllerNvmeNamespaceListResult"),
		},
		"valid request with empty SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_list: %v", "EOF"),
		},
		"valid request with ID mismatch SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":{"name":"","cntlid":0,"Namespaces":null}}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_list: %v", "json response ID mismatch"),
		},
		"valid request with error code from SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"}}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_namespace_list: %v", "json response error: myopierr"),
		},
		"valid request with valid SPDK response": {
			in: testNamespaceName,
			out: &pb.NvmeNamespace{
				Spec: &pb.NvmeNamespaceSpec{
					HostNsid: 22,
				},
				Status: &pb.NvmeNamespaceStatus{
					PciState:     2,
					PciOperState: 1,
				},
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"name": "NvmeEmu0pf1", "cntlid": 1, "Namespaces": [{"nsid": 11, "bdev": "Malloc0", "bdev_type": "spdk", "qn": "", "protocol": ""},{"nsid": 22, "bdev": "Malloc1", "bdev_type": "spdk", "qn": "", "protocol": ""},{"nsid": 13, "bdev": "Malloc2", "bdev_type": "spdk", "qn": "", "protocol": ""}]}}`},
			errCode: codes.OK,
			errMsg:  "",
		},
		"valid request with unknown key": {
			in:      frontend.ResourceIDToNamespaceName(testSubsystemID, "unknown-namespace-id"),
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find key %v", frontend.ResourceIDToNamespaceName(testSubsystemID, "unknown-namespace-id")),
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

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = server.ProtoClone(&testSubsystem)
			testEnv.opiSpdkServer.Subsystems[testSubsystemName].Name = testSubsystemName
			testEnv.opiSpdkServer.Controllers[testControllerName] = server.ProtoClone(&testController)
			testEnv.opiSpdkServer.Controllers[testControllerName].Name = testControllerName
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = server.ProtoClone(&testNamespace)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName].Name = testNamespaceName

			request := &pb.GetNvmeNamespaceRequest{Name: tt.in}
			response, err := testEnv.client.GetNvmeNamespace(testEnv.ctx, request)

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

func TestFrontEnd_StatsNvmeNamespace(t *testing.T) {
	t.Cleanup(checkGlobalTestProtoObjectsNotChanged(t, t.Name()))
	tests := map[string]struct {
		in      string
		out     *pb.VolumeStats
		spdk    []string
		errCode codes.Code
		errMsg  string
	}{
		"valid request with invalid SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":{"controllers":[{"name":"NvmeEmu0pf1","bdevs":[]}]}}`},
			errCode: codes.InvalidArgument,
			errMsg:  fmt.Sprintf("Could not find BdevName: %v", "Malloc1"),
		},
		"valid request with invalid marshal SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result":[]}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_get_iostat: %v", "json: cannot unmarshal array into Go value of type models.NvdaControllerNvmeStatsResult"),
		},
		"valid request with empty SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{""},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_get_iostat: %v", "EOF"),
		},
		"valid request with ID mismatch SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":0,"error":{"code":0,"message":""},"result":{"controllers":[{"name":"NvmeEmu0pf1","bdevs":[]}]}}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_get_iostat: %v", "json response ID mismatch"),
		},
		"valid request with error code from SPDK response": {
			in:      testNamespaceName,
			out:     nil,
			spdk:    []string{`{"id":%d,"error":{"code":1,"message":"myopierr"}}`},
			errCode: codes.Unknown,
			errMsg:  fmt.Sprintf("controller_nvme_get_iostat: %v", "json response error: myopierr"),
		},
		"valid request with valid SPDK response": {
			in: testNamespaceName,
			out: &pb.VolumeStats{
				ReadOpsCount:  12345,
				WriteOpsCount: 54321,
			},
			spdk:    []string{`{"id":%d,"error":{"code":0,"message":""},"result": {"controllers":[{"name":"NvmeEmu0pf1","bdevs":[{"bdev_name":"Malloc0","read_ios":55,"completed_read_ios":55,"write_ios":33,"completed_write_ios":33,"flush_ios":0,"completed_flush_ios":0,"err_read_ios":0,"err_write_ios":0,"err_flush_ios":0},{"bdev_name":"Malloc1","read_ios":12345,"completed_read_ios":12345,"write_ios":54321,"completed_write_ios":54321,"flush_ios":0,"completed_flush_ios":0,"err_read_ios":0,"err_write_ios":0,"err_flush_ios":0}]}]}}`},
			errCode: codes.OK,
			errMsg:  "",
		},
		"valid request with unknown key": {
			in:      "unknown-namespace-id",
			out:     nil,
			spdk:    []string{},
			errCode: codes.NotFound,
			errMsg:  fmt.Sprintf("unable to find key %v", "unknown-namespace-id"),
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

			testEnv.opiSpdkServer.Subsystems[testSubsystemName] = server.ProtoClone(&testSubsystem)
			testEnv.opiSpdkServer.Subsystems[testSubsystemName].Name = testSubsystemName
			testEnv.opiSpdkServer.Controllers[testControllerName] = server.ProtoClone(&testController)
			testEnv.opiSpdkServer.Controllers[testControllerName].Name = testControllerName
			testEnv.opiSpdkServer.Namespaces[testNamespaceName] = server.ProtoClone(&testNamespace)
			testEnv.opiSpdkServer.Namespaces[testNamespaceName].Name = testNamespaceName

			request := &pb.StatsNvmeNamespaceRequest{Name: tt.in}
			response, err := testEnv.client.StatsNvmeNamespace(testEnv.ctx, request)

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
