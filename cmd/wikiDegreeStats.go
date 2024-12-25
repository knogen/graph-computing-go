/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	wikientropy "graph-computing-go/internal/wikiEntropy"

	"github.com/spf13/cobra"
)

// wikiDegreeStatsCmd represents the wikiDegreeStats command
var wikiDegreeStatsCmd = &cobra.Command{
	Use:   "wikiDegreeStats",
	Short: "stats wikipedia degree stats",
	Long:  `stats wikipedia degree stats`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("wikiDegreeStats called")
		wikientropy.GraphDegreeStats()
	},
}

func init() {
	rootCmd.AddCommand(wikiDegreeStatsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// wikiDegreeStatsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// wikiDegreeStatsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
