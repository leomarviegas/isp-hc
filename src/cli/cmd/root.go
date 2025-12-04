//go:build ignore
// +build ignore

package main

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "isp-checker",
	Short: "A CLI tool to diagnose ISP connection issues.",
	Long:  `ISP Health Checker is a comprehensive diagnostics tool to detect instabilities on ISP connections.`,
}

func Execute() error {
	return rootCmd.Execute()
}
