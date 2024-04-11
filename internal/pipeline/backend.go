package pipeline

import "context"

type PipelineState struct {
	Status string
}

type Backend interface {
	GetBuildID(ctx context.Context) (int64, error)
	StartPipeline(ctx context.Context, buildID int64) error
	PutPipeline(ctx context.Context, state *PipelineState) error
	PutPipelineEvent(ctx context.Context, event Event) error
}
