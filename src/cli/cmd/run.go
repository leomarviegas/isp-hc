//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	target     string
	output     string
	mode       string
	simulation string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a health check against a target.",
	Long:  `Runs a comprehensive health check against a target host or IP address.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		result, err := ExecuteRun(ctx, RunOptions{
			Target:         target,
			Mode:           mode,
			OutPath:        output,
			SimulationPath: simulation,
		})
		if err != nil {
			return err
		}
		if err := writeResult(result, output); err != nil {
			return err
		}
		b, _ := result.JSON()
		fmt.Println(string(b))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVar(&target, "target", "8.8.8.8", "Destination to test (e.g. 8.8.8.8).")
	runCmd.Flags().StringVar(&output, "out", "", "Path to save the JSON result.")
	runCmd.Flags().StringVar(&mode, "type", "full", "Probe set to run: ping|dns|traceroute|full.")
	runCmd.Flags().StringVar(&simulation, "simulation", "", "Simulation JSON file path.")
}
