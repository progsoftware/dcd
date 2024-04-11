package pipeline_test

import (
	"fmt"
	"testing"

	"github.com/progsoftware/dcd/internal/pipeline"
)

func TestSingleStepPipelineSuccess(t *testing.T) {
	// Given
	p := pipeline.Pipeline{
		GlobalEnv: map[string]string{
			"GLOBAL_ENV_VAR": "global_env_var_value",
		},
		Steps: []pipeline.Step{
			{
				Name:   "SuccessStep",
				Script: "../../test/step-defs/success/run.sh",
			},
		},
	}

	// When
	eventsChan := p.Run(&pipeline.Metadata{
		Component: "test-component",
		GitSHA:    "test-git-sha",
		BuildID:   "test-build-id",
	})

	// Then
	var events []pipeline.Event
	for event := range eventsChan {
		events = append(events, event)
	}

	// Verify the sequence and content of the events.
	if len(events) < 5 { // Expect Start, StepStart, StepOutput+, StepSuccess, and PipelineSuccess events.
		t.Errorf("Expected at least 5 events, got %d:\n%s", len(events), dumpEvents(events))
	}

	if _, ok := events[0].(pipeline.PipelineStartEvent); !ok {
		t.Errorf("Expected first event to be PipelineStartEvent, got %T", events[0])
	}
	events = events[1:]

	if _, ok := events[0].(pipeline.StepStartEvent); !ok {
		t.Errorf("Expected second event to be StepStartEvent, got %T", events[1])
	}
	events = events[1:]

	outputEvent, ok := events[0].(pipeline.StepOutputEvent)
	if !ok {
		t.Errorf("Expected third event to be StepOutputEvent, got %T", events[2])
	}
	events = events[1:]

	output := outputEvent.Output
	for {
		if e, ok := events[0].(pipeline.StepOutputEvent); ok {
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

	if _, ok := events[0].(pipeline.StepSuccessEvent); !ok {
		t.Errorf("Expected fourth event to be StepSuccessEvent, got %T", events[3])
	}
	events = events[1:]

	// Verify the fifth event is PipelineSuccessEvent.
	if _, ok := events[0].(pipeline.PipelineSuccessEvent); !ok {
		t.Errorf("Expected fifth event to be PipelineSuccessEvent, got %T", events[4])
	}

	// Verify no more events.
	if len(events) > 1 {
		t.Errorf("Expected no more events, got %d:\n%s", len(events), dumpEvents(events))
	}
}

func dumpEvents(events []pipeline.Event) string {
	var dump string
	for i, event := range events {
		dump += dumpEvent(i, event)
	}
	return dump
}

func dumpEvent(i int, event pipeline.Event) string {
	if e, ok := event.(pipeline.StepOutputEvent); ok && e.Output != "" {
		return fmt.Sprintf("%d: %T\n%s", i, event, event.(pipeline.StepOutputEvent).Output)
	}
	if e, ok := event.(pipeline.PipelineFailureEvent); ok && e.Reason != "" {
		return fmt.Sprintf("%d: %T\n%s", i, event, event.(pipeline.PipelineFailureEvent).Reason)
	}
	if e, ok := event.(pipeline.StepFailureEvent); ok && e.Reason != "" {
		return fmt.Sprintf("%d: %T\n%s", i, event, event.(pipeline.StepFailureEvent).Reason)
	}
	return fmt.Sprintf("%d: %T\n", i, event)
}
