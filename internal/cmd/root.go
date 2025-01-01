package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dogdev",
	Short: "An interactive LLM agent CLI",
	Long: `A CLI application that provides an interactive interface to chat with LLM agent
and process files using language models.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
