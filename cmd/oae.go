/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	openalexentropy "graph-computing-go/internal/openAlexEntropy"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// oaeCmd represents the oae command
var oaeCmd = &cobra.Command{
	Use:   "oae",
	Short: "OpenAlex Entropy Analysis",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		_type, _ := cmd.Flags().GetString("type")
		log.Info().Str("Flag", _type).Msg("flag")

		switch _type {
		case "subject":
			openalexentropy.MainSubjectExt()
		case "total":
			openalexentropy.MainExt()
		case "structural":
			openalexentropy.MultilayerSubjectExt()
		case "lv2DistanceComplexity":
			openalexentropy.Lv2DisciplineDistanceComplexity()
		case "test":
			openalexentropy.SubDispolieDistructuralEntropyDemo()
		case "tddc":
			openalexentropy.TopDisciplineDistanceComplexity()
		}

	},
}

func init() {
	rootCmd.AddCommand(oaeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	oaeCmd.PersistentFlags().StringP("type", "t", "subject", "what entropy to calculate, subject or total")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// oaeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
