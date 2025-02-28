/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"graph-computing-go/cmd"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	_ "net/http/pprof"

	"github.com/rs/zerolog"
)

func main() {
	// go func() {
	// 	http.ListenAndServe("0.0.0.0:16060", nil)
	// }()
	// zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	cmd.Execute()
}
