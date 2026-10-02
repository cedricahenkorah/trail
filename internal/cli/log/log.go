/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package log

import (
	"fmt"

	"github.com/spf13/cobra"
)

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Manage project devlog notes",
	Long:  "Add and browse dated Markdown notes for your Trail projects.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("log called")
	},
}

func NewCommand() *cobra.Command {
	logCmd.AddCommand(addCmd)
	return logCmd
}
