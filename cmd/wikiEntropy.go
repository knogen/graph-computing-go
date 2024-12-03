/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	wikientropy "graph-computing-go/internal/wikiEntropy"

	"github.com/spf13/cobra"
)

// wikiEntropyCmd represents the wikiEntropy command
var wikiEntropyCmd = &cobra.Command{
	Use:   "wikiEntropy",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("wikiEntropy called")
		wikientropy.Main()
	},
}

func init() {
	rootCmd.AddCommand(wikiEntropyCmd)

}
