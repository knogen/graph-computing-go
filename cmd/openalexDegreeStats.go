/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	openalexentropy "graph-computing-go/internal/openAlexEntropy"

	"github.com/spf13/cobra"
)

// openalexDegreeStatsCmd represents the openalexDegreeStats command
var openalexDegreeStatsCmd = &cobra.Command{
	Use:   "openalexDegreeStats",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("openalexDegreeStats called")
		openalexentropy.GraphDegreeStats()
	},
}

func init() {
	rootCmd.AddCommand(openalexDegreeStatsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// openalexDegreeStatsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// openalexDegreeStatsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
