/*
Copyright 2014-2017 Bo Blanton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Main RPC server endpoint

package server

import (
	"cadent/server/utils"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"cadent/server/writers/rpc/pb"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	logging "gopkg.in/op/go-logging.v1"
	"net"
	"strings"
)

type Server struct {
	Listen      string
	CertKeyFile string
	CertFile    string

	Indexer indexer.Indexer
	Metrics metrics.Metrics
	Tracer  opentracing.Tracer

	log       *logging.Logger
	startstop utils.StartStop

	ListenAddr net.Addr
	gRPCserver *grpc.Server
	shutdown   chan bool
}

func New() (*Server, error) {
	s := new(Server)
	s.log = logging.MustGetLogger("api.grpc")
	s.shutdown = make(chan bool)
	return s, nil
}

func (s *Server) runServer(lis net.Listener) {
	go func() {

		err := s.gRPCserver.Serve(lis)
		// the error use of closed network connection is what happens on "shutdown" so no need to err
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				s.log.Fatalf("Failed to start gRPC server: %v", err)
			} else {
				s.log.Warningf("gRPC server Shutdown on %v", lis.Addr().String())
			}
		}
		s.log.Noticef("Started gRPC server on %v", lis.Addr().String())
	}()

	<-s.shutdown
	if s.gRPCserver == nil {
		return
	}
	s.log.Warning("Shutting down gRPC server")
	s.gRPCserver.GracefulStop()
	s.gRPCserver = nil
}

func (s *Server) Start() {
	s.startstop.Start(func() {
		var opts []grpc.ServerOption

		lis, err := net.Listen("tcp", s.Listen)
		if err != nil {
			s.log.Fatalf("Failed bind to listen (%s) %v", s.Listen, err)
		}
		s.ListenAddr = lis.Addr()

		if len(s.CertKeyFile) > 0 {
			creds, err := credentials.NewServerTLSFromFile(s.CertFile, s.CertKeyFile)
			if err != nil {
				s.log.Fatalf("Failed to generate credentials %v", err)
			}
			opts = []grpc.ServerOption{grpc.Creds(creds)}
		}

		// XXX TODO: once otgrpc has Streaming Interceptors in the mix add this here
		if s.Tracer != nil {
			opts = append(opts,
				grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(s.Tracer)),
			)
		}

		s.gRPCserver = grpc.NewServer(opts...)
		mServer := new(MetricServer)
		mServer.Indexer = s.Indexer
		mServer.Metrics = s.Metrics
		mServer.Tracer = s.Tracer

		pb.RegisterCadentMetricServer(s.gRPCserver, mServer)
		reflection.Register(s.gRPCserver)
		go s.runServer(lis)

	})
}

func (s *Server) Stop() {
	s.startstop.Stop(func() {
		s.shutdown <- true
	})
}
