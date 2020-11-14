package workflow

import "bytes"

type CommandResult struct {
	ContainerID string
	Command     []string
	StdOut      bytes.Buffer
	StdErr      bytes.Buffer
	ExitCode    int
	Error       error
}

type JobResult struct {
	Commands []CommandResult
	Error    error
}

func (r *JobResult) AddCommandResult(command CommandResult) {
	r.Commands = append(r.Commands, command)
}
