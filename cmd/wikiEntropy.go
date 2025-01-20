/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	wikientropy "graph-computing-go/internal/wikiEntropy"

	"github.com/rs/zerolog/log"
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
		_type, _ := cmd.Flags().GetString("type")
		log.Info().Str("Flag", _type).Msg("flag")

		switch _type {
		case "subject":
			wikientropy.MainSubject()
		case "total":
			wikientropy.Main()
		case "structural":
			wikientropy.MultilayerSubjectExt()
		}

	},
}

func init() {
	rootCmd.AddCommand(wikiEntropyCmd)

	wikiEntropyCmd.PersistentFlags().StringP("type", "t", "subject", "what entropy to calculate, subject or total")
}
