package dcd_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/progsoftware/dcd/internal/dcd"
)

type MockBackend struct {
	BuildID int64
}

func (b *MockBackend) GetBuildID(ctx context.Context) (int64, error) {
	b.BuildID++
	return b.BuildID, nil
}

func (b *MockBackend) StartPipeline(ctx context.Context, buildID int64) error {
	return nil
}

func (b *MockBackend) PutPipeline(ctx context.Context, state *dcd.PipelineState) error {
	return nil
}

func (b *MockBackend) PutPipelineEvent(ctx context.Context, event dcd.Event) error {
	return nil
}

func TestSingleStepPipelineSuccess(t *testing.T) {
	// Given
	pipeline := dcd.NewPipeline()
	pipeline.SetMetadata(&dcd.Metadata{
		Component: "test-component",
		GitSHA:    "test-git-sha",
	})
	pipeline.SetDefinition(&dcd.PipelineDefinition{
		GlobalEnv: map[string]string{
			"GLOBAL_ENV_VAR": "global_env_var_value",
		},
		Steps: []dcd.Step{
			{
				Name:   "SuccessStep",
				Script: "../../test/step-defs/success/run.sh",
			},
		},
	})
	pipeline.SetBackend(&MockBackend{})

	// When
	eventsChan, err := pipeline.Run()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Then
	var events []dcd.Event
	for event := range eventsChan {
		events = append(events, event)
	}

	// Verify the sequence and content of the events.
	if len(events) < 5 { // Expect Start, StepStart, StepOutput+, StepSuccess, and PipelineSuccess events.
		t.Errorf("Expected at least 5 events, got %d:\n%s", len(events), dumpEvents(events))
	}

	if _, ok := events[0].(dcd.PipelineStartEvent); !ok {
		t.Errorf("Expected first event to be PipelineStartEvent, got %T", events[0])
	}
	events = events[1:]

	if _, ok := events[0].(dcd.StepStartEvent); !ok {
		t.Errorf("Expected second event to be StepStartEvent, got %T", events[1])
	}
	events = events[1:]

	outputEvent, ok := events[0].(dcd.StepOutputEvent)
	if !ok {
		t.Errorf("Expected third event to be StepOutputEvent, got %T", events[2])
	}
	events = events[1:]

	output := outputEvent.Output
	for {
		if e, ok := events[0].(dcd.StepOutputEvent); ok {
			output += e.Output
			events = events[1:]
		} else {
			break
		}
	}

	expectedOutput := "component: test-component\ngit sha: test-git-sha\nbuild id: test-build-id\nglobal env var: global_env_var_value\noutput to stdout 1\noutput to stderr 1\noutput to stdout 2\noutput to stderr 2\n"
	if output != expectedOutput {
		t.Errorf("Expected output to be %q, got %q", expectedOutput, output)
	}

	if _, ok := events[0].(dcd.StepSuccessEvent); !ok {
		t.Errorf("Expected fourth event to be StepSuccessEvent, got %T", events[3])
	}
	events = events[1:]

	// Verify the fifth event is PipelineSuccessEvent.
	if _, ok := events[0].(dcd.PipelineSuccessEvent); !ok {
		t.Errorf("Expected fifth event to be PipelineSuccessEvent, got %T", events[4])
	}

	// Verify no more events.
	if len(events) > 1 {
		t.Errorf("Expected no more events, got %d:\n%s", len(events), dumpEvents(events))
	}
}

func dumpEvents(events []dcd.Event) string {
	var dump string
	for i, event := range events {
		dump += dumpEvent(i, event)
	}
	return dump
}

func dumpEvent(i int, event dcd.Event) string {
	if e, ok := event.(dcd.StepOutputEvent); ok && e.Output != "" {
		return fmt.Sprintf("%d: %T\n%s", i, event, e.Output)
	}
	if e, ok := event.(dcd.PipelineFailureEvent); ok && e.Reason != "" {
		return fmt.Sprintf("%d: %T\n%s", i, event, e.Reason)
	}
	if e, ok := event.(dcd.StepFailureEvent); ok && e.Reason != "" {
		return fmt.Sprintf("%d: %T\n%s", i, event, e.Reason)
	}
	return fmt.Sprintf("%d: %T\n", i, event)
}
