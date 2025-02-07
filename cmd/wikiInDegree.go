/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	wikipediaindegree "graph-computing-go/internal/wikipediaInDegree"

	"github.com/spf13/cobra"
)

// wikiInDegreeCmd represents the wikiInDegree command
var wikiInDegreeCmd = &cobra.Command{
	Use:   "wikiInDegree",
	Short: "calculates the in degree of a wiki page",
	Long:  `calculates the in degree of a wiki page`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("wikiInDegree called")
		wikipediaindegree.Main()
	},
}

func init() {
	rootCmd.AddCommand(wikiInDegreeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// wikiInDegreeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// wikiInDegreeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
