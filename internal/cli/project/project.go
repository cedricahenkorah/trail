/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package project

import (
	"fmt"

	"github.com/spf13/cobra"
)

// projectCmd represents the project command
var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects tracked by Trail",
	Long: `Create and manage the projects you track with Trail. Project notes,
	tasks, and issues are stored in Trail's data directory. A project can
	optionally link to a local working directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("project called")
	},
}

func NewCommand() *cobra.Command {
	projectCmd.AddCommand(addCmd)
	projectCmd.AddCommand(listCmd)
	return projectCmd
}
