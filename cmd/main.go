// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2022-2023 Dell Inc, or its subsidiaries.
// Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// Copyright (C) 2023 Intel Corporation

// main is the main package of the application
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/opiproject/gospdk/spdk"

	fe "github.com/opiproject/opi-nvidia-bridge/pkg/frontend"
	"github.com/opiproject/opi-smbios-bridge/pkg/inventory"
	"github.com/opiproject/opi-spdk-bridge/pkg/backend"
	"github.com/opiproject/opi-spdk-bridge/pkg/frontend"
	"github.com/opiproject/opi-spdk-bridge/pkg/middleend"
	"github.com/opiproject/opi-spdk-bridge/pkg/utils"
	"github.com/opiproject/opi-strongswan-bridge/pkg/ipsec"

	pc "github.com/opiproject/opi-api/inventory/v1/gen/go"
	ps "github.com/opiproject/opi-api/security/v1/gen/go"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/redis"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func main() {
	var grpcPort int
	flag.IntVar(&grpcPort, "grpc_port", 50051, "The gRPC server port")

	var httpPort int
	flag.IntVar(&httpPort, "http_port", 8082, "The HTTP server port")

	var spdkAddress string
	flag.StringVar(&spdkAddress, "spdk_addr", "/var/tmp/spdk.sock", "Points to SPDK unix socket/tcp socket to interact with")

	var tlsFiles string
	flag.StringVar(&tlsFiles, "tls", "", "TLS files in server_cert:server_key:ca_cert format.")

	var redisAddress string
	flag.StringVar(&redisAddress, "redis_addr", "127.0.0.1:6379", "Redis address in ip_address:port format")

	flag.Parse()

	// Create KV store for persistence
	options := redis.DefaultOptions
	options.Address = redisAddress
	options.Codec = utils.ProtoCodec{}
	store, err := redis.NewClient(options)
	if err != nil {
		log.Panic(err)
	}
	defer func(store gokv.Store) {
		err := store.Close()
		if err != nil {
			log.Panic(err)
		}
	}(store)

	go runGatewayServer(grpcPort, httpPort)
	runGrpcServer(grpcPort, spdkAddress, tlsFiles, store)
}

func runGrpcServer(grpcPort int, spdkAddress string, tlsFiles string, store gokv.Store) {
	tp := utils.InitTracerProvider("opi-nvidia-bridge")
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Panicf("Tracer Provider Shutdown: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Panicf("failed to listen: %v", err)
	}

	jsonRPC := spdk.NewClient(spdkAddress)
	frontendOpiNvidiaServer := fe.NewServer(jsonRPC, store)
	frontendOpiSpdkServer := frontend.NewServer(jsonRPC, store)
	backendOpiSpdkServer := backend.NewServer(jsonRPC, store)
	middleendOpiSpdkServer := middleend.NewServer(jsonRPC, store)

	var serverOptions []grpc.ServerOption
	if tlsFiles == "" {
		log.Println("TLS files are not specified. Use insecure connection.")
	} else {
		log.Println("Use TLS certificate files:", tlsFiles)
		config, err := utils.ParseTLSFiles(tlsFiles)
		if err != nil {
			log.Panic("Failed to parse string with tls paths:", err)
		}
		log.Println("TLS config:", config)
		var option grpc.ServerOption
		if option, err = utils.SetupTLSCredentials(config); err != nil {
			log.Panic("Failed to setup TLS:", err)
		}
		serverOptions = append(serverOptions, option)
	}
	serverOptions = append(serverOptions,
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(
			logging.UnaryServerInterceptor(utils.InterceptorLogger(log.Default()),
				logging.WithLogOnEvents(
					logging.StartCall,
					logging.FinishCall,
					logging.PayloadReceived,
					logging.PayloadSent,
				),
			)),
	)
	s := grpc.NewServer(serverOptions...)

	pb.RegisterFrontendNvmeServiceServer(s, frontendOpiNvidiaServer)
	pb.RegisterFrontendVirtioBlkServiceServer(s, frontendOpiNvidiaServer)
	pb.RegisterFrontendVirtioScsiServiceServer(s, frontendOpiSpdkServer)
	pb.RegisterNvmeRemoteControllerServiceServer(s, backendOpiSpdkServer)
	pb.RegisterNullVolumeServiceServer(s, backendOpiSpdkServer)
	pb.RegisterMallocVolumeServiceServer(s, backendOpiSpdkServer)
	pb.RegisterAioVolumeServiceServer(s, backendOpiSpdkServer)
	pb.RegisterMiddleendEncryptionServiceServer(s, middleendOpiSpdkServer)
	pc.RegisterInventoryServiceServer(s, &inventory.Server{})
	ps.RegisterIPsecServiceServer(s, &ipsec.Server{})

	reflection.Register(s)

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
}

func runGatewayServer(grpcPort int, httpPort int) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	endpoint := fmt.Sprintf("localhost:%d", grpcPort)
	registerGatewayHandler(ctx, mux, endpoint, opts, pc.RegisterInventoryServiceHandlerFromEndpoint, "inventory")

	registerGatewayHandler(ctx, mux, endpoint, opts, pb.RegisterAioVolumeServiceHandlerFromEndpoint, "backend aio")
	registerGatewayHandler(ctx, mux, endpoint, opts, pb.RegisterNullVolumeServiceHandlerFromEndpoint, "backend null")
	registerGatewayHandler(ctx, mux, endpoint, opts, pb.RegisterMallocVolumeServiceHandlerFromEndpoint, "backend malloc")
	registerGatewayHandler(ctx, mux, endpoint, opts, pb.RegisterNvmeRemoteControllerServiceHandlerFromEndpoint, "backend nvme")

	registerGatewayHandler(ctx, mux, endpoint, opts, pb.RegisterMiddleendEncryptionServiceHandlerFromEndpoint, "middleend encryption")
	registerGatewayHandler(ctx, mux, endpoint, opts, pb.RegisterMiddleendQosVolumeServiceHandlerFromEndpoint, "middleend qos")

	registerGatewayHandler(ctx, mux, endpoint, opts, pb.RegisterFrontendVirtioBlkServiceHandlerFromEndpoint, "frontend virtio-blk")
	registerGatewayHandler(ctx, mux, endpoint, opts, pb.RegisterFrontendVirtioScsiServiceHandlerFromEndpoint, "frontend virtio-scsi")
	registerGatewayHandler(ctx, mux, endpoint, opts, pb.RegisterFrontendNvmeServiceHandlerFromEndpoint, "frontend nvme")

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	log.Printf("HTTP Server listening at %v", httpPort)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", httpPort),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Panic("cannot start HTTP gateway server")
	}
}

type registerHandlerFunc func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

func registerGatewayHandler(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption, registerFunc registerHandlerFunc, serviceName string) {
	err := registerFunc(ctx, mux, endpoint, opts)
	if err != nil {
		log.Panicf("cannot register %s handler server: %v", serviceName, err)
	}
}
