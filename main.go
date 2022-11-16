// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"plugin"

	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

func main() {
	flag.Parse()
	// Load the plugin
	plug, err := plugin.Open("/opi-nvidia-bridge.so")
	if err != nil {
		log.Fatal(err)
	}
	// 2. Look for an exported symbol such as a function or variable
	feNvmeSymbol, err := plug.Lookup("PluginFrontendNvme")
	if err != nil {
		log.Fatal(err)
	}
	// 3. Attempt to cast the symbol to the Shipper
	var feNvme pb.FrontendNvmeServiceServer
	feNvme, ok := feNvmeSymbol.(pb.FrontendNvmeServiceServer)
	if !ok {
		log.Fatal("Invalid feNvme type")
	}
	log.Printf("plugin serevr is %v", feNvme)
	// 4. If everything is ok from the previous assertions, then we can proceed
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	pb.RegisterFrontendNvmeServiceServer(s, feNvme)
	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
