package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"

	"../../../proto" // Adjust path
)

var (
	names string // For client stream
)

var streamCmd = &cobra.Command{
	Use:   "stream [server|client|bi]",
	Short: "Interact with streaming RPCs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conn, c := connect()
		defer conn.Close()

		token := generateToken(role)
		ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+token)

		switch args[0] {
		case "server":
			doServerStream(c, ctx)
		case "client":
			doClientStream(c, ctx)
		case "bi":
			doBiStream(c, ctx)
		default:
			log.Fatalf("Invalid stream type: %s", args[0])
		}
	},
}

func init() {
	streamCmd.Flags().StringVarP(&name, "name", "n", "Stream World", "Name for server stream")
	streamCmd.Flags().StringVarP(&names, "names", "s", "Alice,Bob", "Comma-separated names for client/bi stream")
	streamCmd.Flags().StringVarP(&role, "role", "r", "admin", "Role: user or admin (streaming often admin-only)")
}

func doServerStream(c proto.AuthServiceClient, ctx context.Context) {
	stream, err := c.GreetManyTimes(ctx, &proto.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("Server stream error: %v", err)
	}
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Recv error: %v", err)
		}
		fmt.Println("Server Stream:", msg.Message)
	}
}

func doClientStream(c proto.AuthServiceClient, ctx context.Context) {
	clientStream, err := c.LongGreeting(ctx)
	if err != nil {
		log.Fatalf("Client stream error: %v", err)
	}
	for _, n := range strings.Split(names, ",") {
		clientStream.Send(&proto.HelloRequest{Name: n})
	}
	reply, err := clientStream.CloseAndRecv()
	if err != nil {
		log.Fatalf("Close error: %v", err)
	}
	fmt.Println("Client Stream Reply:", reply.Message)
}

func doBiStream(c proto.AuthServiceClient, ctx context.Context) {
	biStream, err := c.EveryoneGreeting(ctx)
	if err != nil {
		log.Fatalf("Bi stream error: %v", err)
	}
	done := make(chan bool)
	go func() {
		for _, n := range strings.Split(names, ",") {
			biStream.Send(&proto.HelloRequest{Name: n})
			time.Sleep(500 * time.Millisecond)
		}
		biStream.CloseSend()
	}()
	go func() {
		for {
			msg, err := biStream.Recv()
			if err == io.EOF {
				done <- true
				break
			}
			if err != nil {
				log.Printf("Recv error: %v", err)
				done <- true
				break
			}
			fmt.Println("Bi Stream:", msg.Message)
		}
	}()
	<-done
}
