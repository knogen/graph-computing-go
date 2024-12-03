/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	extractwikipediadump "graph-computing-go/internal/extractWikipediadump"

	"github.com/spf13/cobra"
)

// excractWikipediaDumpCmd represents the excractWikipediaDump command
var excractWikipediaDumpCmd = &cobra.Command{
	Use:   "excractWikipediaDump",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("excractWikipediaDump called")
		extractwikipediadump.ExtractWikipediaDump()
	},
}

func init() {
	rootCmd.AddCommand(excractWikipediaDumpCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// excractWikipediaDumpCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// excractWikipediaDumpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
