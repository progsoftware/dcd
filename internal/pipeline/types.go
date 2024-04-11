package pipeline

import (
	"fmt"
	"time"
)

type Metadata struct {
	Component string
	GitSHA    string
	BuildID   string
}

// Step represents a single step in the pipeline.
type Step struct {
	Name   string `yaml:"name"`
	Script string `yaml:"script"`
}

// Pipeline represents the structure of the pipeline YAML.
type Pipeline struct {
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
	BuildID string
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