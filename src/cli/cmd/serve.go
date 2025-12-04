//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serveAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run a Prometheus metrics endpoint.",
	Long:  `Exposes a /metrics endpoint for Prometheus to scrape.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Starting metrics server on %s/metrics\n", serveAddr)
		return startMetricsServer(serveAddr)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringVar(&serveAddr, "addr", ":9100", "Address for the metrics server.")
}
