package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "raft",
		Short: "Simple Raft server",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
