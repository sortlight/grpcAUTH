package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"./proto"
)


var jwtSecret = []byte("supersecretkey")

func main() {
	
	creds, err := credentials.NewClientTLSFromFile("certs/server.crt", "localhost")
	if err != nil {
		log.Fatalf("Failed to load certs: %v", err)
	}

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	c := proto.NewAuthServiceClient(conn)

	
	genToken := func(role string) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"role": role,
			"exp":  time.Now().Add(time.Hour).Unix(),
		})
		tokenStr, _ := token.SignedString(jwtSecret)
		return tokenStr
	}

	
	ctxUser := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+genToken("user"))
	r, err := c.SayHello(ctxUser, &proto.HelloRequest{Name: "User World"})
	if err != nil {
		log.Printf("Unary user error: %v", err)
	} else {
		fmt.Println("Unary User Greeting:", r.Message)
	}

	
	stream, err := c.GreetManyTimes(ctxUser, &proto.HelloRequest{Name: "Stream World"})
	if err != nil {
		log.Printf("Server stream error: %v", err)
	}
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Server stream recv error: %v", err)
			break
		}
		fmt.Println("Server Stream:", msg.Message)
	}

	
	ctxAdmin := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+genToken("admin"))
	clientStream, err := c.LongGreeting(ctxAdmin)
	if err != nil {
		log.Fatalf("Client stream error: %v", err)
	}
	names := []string{"Alice", "Bob", "Charlie"}
	for _, name := range names {
		clientStream.Send(&proto.HelloRequest{Name: name})
	}
	reply, err := clientStream.CloseAndRecv()
	if err != nil {
		log.Printf("Client stream close error: %v", err)
	} else {
		fmt.Println("Client Stream Reply:", reply.Message)
	}

	// Bidirectional as admin
	biStream, err := c.EveryoneGreeting(ctxAdmin)
	if err != nil {
		log.Fatalf("Bi stream error: %v", err)
	}
	go func() {
		biNames := []string{"Dave", "Eve", "Frank"}
		for _, name := range biNames {
			biStream.Send(&proto.HelloRequest{Name: name})
			time.Sleep(500 * time.Millisecond)
		}
		biStream.CloseSend()
	}()
	for {
		msg, err := biStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Bi stream recv error: %v", err)
			break
		}
		fmt.Println("Bi Stream:", msg.Message)
	}


	_, err = c.LongGreeting(ctxUser)
	if err != nil {
		fmt.Println("Expected RBAC denial:", err)
	}
}