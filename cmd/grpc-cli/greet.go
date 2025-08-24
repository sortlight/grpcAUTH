package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"././grpcAUTH/proto" // Adjust to your path, e.g., "github.com/yourusername/grpc-auth-demo/proto"
)

var (
	name string
	role string
)

var greetCmd = &cobra.Command{
	Use:   "greet",
	Short: "Send a unary greet to the server",
	Run: func(cmd *cobra.Command, args []string) {
		conn, c := connect()
		defer conn.Close()

		token := generateToken(role)
		ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

		r, err := c.SayHello(ctx, &proto.HelloRequest{Name: name})
		if err != nil {
			log.Fatalf("Greet error: %v", err)
		}
		fmt.Println("Greeting:", r.Message)
	},
}

func init() {
	greetCmd.Flags().StringVarP(&name, "name", "n", "World", "Name to greet")
	greetCmd.Flags().StringVarP(&role, "role", "r", "user", "Role: user or admin")
}

func connect() (*grpc.ClientConn, proto.AuthServiceClient) {
	creds, err := credentials.NewClientTLSFromFile("../../../certs/server.crt", "localhost") // Adjust path
	if err != nil {
		log.Fatalf("Certs error: %v", err)
	}
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Connect error: %v", err)
	}
	return conn, proto.NewAuthServiceClient(conn)
}

func generateToken(role string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"role": role, "exp": time.Now().Add(time.Hour).Unix()})
	tokenStr, _ := token.SignedString([]byte("supersecretkey"))
	return tokenStr
}
