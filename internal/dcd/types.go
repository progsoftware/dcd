package dcd

import (
	"fmt"
	"strings"
	"time"
)

type Metadata struct {
	Component string
	GitSHA    string
}

// Step represents a single step in the pipeline.
type Step struct {
	Name   string `yaml:"name"`
	Script string `yaml:"script"`
}

// PipelineState represents the state of a pipeline at a point in time as serialised.
type PipelineState struct {
	BuildID int64
	Status  string
}

// Pipeline represents a pipeline that you run
type Pipeline struct {
	definition *PipelineDefinition
	metadata   *Metadata
	backend    Backend
}

// PipelineDefinition represents the structure of the pipeline YAML.
type PipelineDefinition struct {
	GlobalEnv map[string]string `yaml:"global-env"`
	Steps     []Step            `yaml:"steps"`
}

type Event interface {
	Timestamp() time.Time
	LogMessage() string // A method to generate a log message specific to the event type
}

type BaseEvent struct {
	EventTime time.Time
}

func (be BaseEvent) Timestamp() time.Time {
	return be.EventTime
}

// PipelineStartEvent signifies the start of the pipeline execution.
type PipelineStartEvent struct {
	BaseEvent
	BuildID int64
}

func (p PipelineStartEvent) LogMessage() string {
	return "Pipeline start"
}

// PipelineSuccessEvent signifies the successful completion of the pipeline.
type PipelineSuccessEvent struct {
	BaseEvent
}

func (p PipelineSuccessEvent) LogMessage() string {
	return "Pipeline succeeded"
}

// PipelineFailureEvent signifies a failure in pipeline execution.
type PipelineFailureEvent struct {
	BaseEvent
	Reason string
}

func (p PipelineFailureEvent) LogMessage() string {
	return fmt.Sprintf("Pipeline failed: %s", p.Reason)
}

// StepStartEvent signifies the start of a pipeline step.
type StepStartEvent struct {
	BaseEvent
	StepName string
}

func (s StepStartEvent) LogMessage() string {
	return fmt.Sprintf("Step started: %s", s.StepName)
}

// StepOutputEvent is used to log output from a pipeline step.
type StepOutputEvent struct {
	BaseEvent
	StepName string
	Output   string
}

func (s StepOutputEvent) LogMessage() string {
	return fmt.Sprintf("Output from %s: %s", s.StepName, s.Output)
}

// StepSuccessEvent signifies the successful completion of a pipeline step.
type StepSuccessEvent struct {
	BaseEvent
	StepName string
}

func (s StepSuccessEvent) LogMessage() string {
	return fmt.Sprintf("Step succeeded: %s", s.StepName)
}

// StepFailureEvent signifies a failure in a pipeline step.
type StepFailureEvent struct {
	BaseEvent
	StepName string
	Reason   string
}

func (s StepFailureEvent) LogMessage() string {
	return fmt.Sprintf("Step failed: %s, Reason: %s", s.StepName, s.Reason)
}

type UncommittedChangesError struct{}

func (e UncommittedChangesError) Error() string {
	return "the working directory contains uncommitted changes"
}

type UnsyncedChangesError struct {
	RemoteAhead int
	LocalAhead  int
}

func (e UnsyncedChangesError) Error() string {
	parts := []string{}
	if e.LocalAhead == 1 {
		parts = append(parts, "1 local commit")
	} else if e.LocalAhead > 1 {
		parts = append(parts, fmt.Sprintf("%d local commits", e.LocalAhead))
	}
	if e.RemoteAhead == 1 {
		parts = append(parts, "1 remote commit")
	} else if e.RemoteAhead > 1 {
		parts = append(parts, fmt.Sprintf("%d remote commits", e.RemoteAhead))
	}
	return fmt.Sprintf("the local repository is out of sync with the upstream repository: %s", strings.Join(parts, " and "))
}

type NotOnMainBranchError struct {
	Branch string
}

func (e NotOnMainBranchError) Error() string {
	return fmt.Sprintf("the current branch is %q, not main", e.Branch)
}

type NotTrackingOriginMainError struct {
	Output string
}

func (e NotTrackingOriginMainError) Error() string {
	return fmt.Sprintf("the current branch is not tracking origin/main: %s", e.Output)
}
