package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "grpc-cli",
	Short: "CLI tool to interact with gRPC Auth Demo server",
	Long:  `A versatile CLI built with Cobra to call unary and streaming RPCs on the gRPC server.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(greetCmd)
	rootCmd.AddCommand(streamCmd)
}
