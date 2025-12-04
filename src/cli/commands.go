package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func parseAndRun() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("Usage: isp-checker [run|serve] [flags]")
	}
	ctx := context.Background()
	switch os.Args[1] {
	case "run":
		return runCommand(ctx, os.Args[2:])
	case "serve":
		return serveCommand(ctx, os.Args[2:])
	default:
		return fmt.Errorf("unknown command %s", os.Args[1])
	}
}

func runCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	target := fs.String("target", "8.8.8.8", "target host or IP")
	runType := fs.String("type", "full", "probe type: ping|dns|traceroute|full")
	outPath := fs.String("out", "", "write result JSON to file")
	simPath := fs.String("simulation", "", "simulation JSON file (uses scenario instead of live probes)")
	fs.Parse(args)

	opts := RunOptions{
		Target:         *target,
		Mode:           *runType,
		OutPath:        *outPath,
		SimulationPath: *simPath,
	}

	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	result, err := ExecuteRun(ctx, opts)
	if err != nil {
		return fmt.Errorf("run failed: %w", err)
	}

	if err := writeResult(result, *outPath); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	b, _ := result.JSON()
	os.Stdout.Write(b)
	return nil
}

func serveCommand(_ context.Context, args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":9100", "address for metrics endpoint")
	fs.Parse(args)

	log.Printf("starting metrics server on %s/metrics", *addr)
	return startMetricsServer(*addr)
}
