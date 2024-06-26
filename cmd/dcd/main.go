package main

import (
	"fmt"
	"os"

	dcd "github.com/progsoftware/dcd/internal/dcd"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "image-usage-message" {
		fmt.Println("This image should be used as a base image, not run directly - see README.md for more information.")
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: dcd <command> [args...]")
		os.Exit(1)
	}
	pipeline := dcd.NewPipeline()
	if err := pipeline.LoadMetadata(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	command := os.Args[1]
	switch command {
	case "run":
		runPipeline(pipeline, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func runPipeline(pipeline *dcd.Pipeline, args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: dcd run <pipeline-file>")
		os.Exit(1)
	}
	filename := args[0]
	if err := pipeline.LoadPipelineDefinition(filename); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	eventsChan, err := pipeline.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for event := range eventsChan {
		fmt.Println(event.LogMessage())
		if _, ok := event.(dcd.PipelineSuccessEvent); ok {
			os.Exit(0)
		}
		if _, ok := event.(dcd.PipelineFailureEvent); ok {
			os.Exit(1)
		}
	}
}
