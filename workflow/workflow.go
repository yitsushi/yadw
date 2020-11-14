package workflow

import (
	"context"

	"github.com/docker/docker/client"
)

// Workflow is a series of Jobs.
type Workflow struct {
	Jobs   []*Job
	client *client.Client
}

// NewWorkflow creates a new workflow.
func NewWorkflow(cli *client.Client) Workflow {
	return Workflow{
		client: cli,
		Jobs:   []*Job{},
	}
}

func (workflow *Workflow) AddJob(job *Job) {
	workflow.Jobs = append(workflow.Jobs, job)
}

// Execute the workflow.
func (workflow *Workflow) Execute(ctx context.Context) {
	for _, job := range workflow.Jobs {
		job.Run(ctx, workflow.client)
	}
}
