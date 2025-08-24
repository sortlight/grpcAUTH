package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"./proto"
)


var jwtSecret = []byte("supersecretkey")


var allowedRoles = map[string][]string{
	"/auth.AuthService/SayHello":        {"user", "admin"},
	"/auth.AuthService/GreetManyTimes":  {"user", "admin"},
	"/auth.AuthService/LongGreeting":    {"admin"},
	"/auth.AuthService/EveryoneGreeting": {"admin"},
}

type server struct {
	proto.UnimplementedAuthServiceServer
}

func (s *server) SayHello(ctx context.Context, in *proto.HelloRequest) (*proto.HelloReply, error) {
	return &proto.HelloReply{Message: fmt.Sprintf("Hello, %s!", in.Name)}, nil
}

func (s *server) GreetManyTimes(in *proto.HelloRequest, stream proto.AuthService_GreetManyTimesServer) error {
	for i := 0; i < 5; i++ {
		if err := stream.Send(&proto.HelloReply{Message: fmt.Sprintf("Hello %s! (%d)", in.Name, i+1)}); err != nil {
			return err
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

func (s *server) LongGreeting(stream proto.AuthService_LongGreetingServer) (*proto.HelloReply, error) {
	var names []string
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		names = append(names, req.Name)
	}
	return &proto.HelloReply{Message: fmt.Sprintf("Hello to everyone: %v!", names)}, stream.Context().Err()
}

func (s *server) EveryoneGreeting(stream proto.AuthService_EveryoneGreetingServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&proto.HelloReply{Message: fmt.Sprintf("Echo: Hello %s!", req.Name)}); err != nil {
			return err
		}
	}
}


func authInterceptor(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "missing authorization token")
	}

	tokenStr := values[0]
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "invalid claims")
	}
	role, ok := claims["role"].(string)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "missing role")
	}

	// RBAC check (get method from context or use auth.NewAuthFunc if needed)
	// For simplicity, assume we chain with selector or use info in unary/stream
	// Note: For full RBAC, use middleware selector or custom

	return metadata.AppendToOutgoingContext(ctx, "role", role), nil  // Pass role downstream if needed
}


func rbacUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	
	md, _ := metadata.FromIncomingContext(ctx)
	role := md.Get("role")
	if len(role) == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "no role found")
	}

	allowed, ok := allowedRoles[info.FullMethod]
	if !ok || !contains(allowed, role[0]) {
		return nil, status.Errorf(codes.PermissionDenied, "role %s not allowed for %s", role[0], info.FullMethod)
	}

	return handler(ctx, req)
}


func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}


func rbacStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.ServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()
	md, _ := metadata.FromIncomingContext(ctx)
	role := md.Get("role")
	if len(role) == 0 {
		return status.Errorf(codes.PermissionDenied, "no role found")
	}

	allowed, ok := allowedRoles[info.FullMethod]
	if !ok || !contains(allowed, role[0]) {
		return status.Errorf(codes.PermissionDenied, "role %s not allowed for %s", role[0], info.FullMethod)
	}

	return handler(srv, ss)
}

func main() {
	
	metrics := prometheus.NewServerMetrics(
		prometheus.WithServerHandlingTimeHistogram(),
	)
	reg := prometheus.NewRegistry() 
	reg.MustRegister(metrics)

	
	creds, err := credentials.NewServerTLSFromFile("certs/server.crt", "certs/server.key")
	if err != nil {
		log.Fatalf("Failed to load certs: %v", err)
	}

	
	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			metrics.UnaryServerInterceptor(),
			auth.UnaryServerInterceptor(authInterceptor),
			rbacUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			metrics.StreamServerInterceptor(),
			auth.StreamServerInterceptor(authInterceptor),
			rbacStreamInterceptor,
		),
	}

	s := grpc.NewServer(opts...)
	proto.RegisterAuthServiceServer(s, &server{})

	// Start gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	go func() {
		log.Println("gRPC server starting on :50051")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Println("Prometheus metrics on :2112/metrics")
	if err := http.ListenAndServe(":2112", nil); err != nil {
		log.Fatalf("Failed to serve metrics: %v", err)
	}
}