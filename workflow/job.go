package workflow

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

type jobContext struct {
	Context     context.Context
	Client      *docker.Client
	ContainerID string
}

// Job is a single job executed in a workflow.
type Job struct {
	Name        string
	Commands    [][]string
	Image       string
	StopOnError bool
	Environment []string

	Result *JobResult
}

func (job *Job) createContainer(ctx *jobContext) error {
	cont, err := ctx.Client.ContainerCreate(
		ctx.Context,
		&container.Config{
			Env:          job.Environment,
			Image:        job.Image,
			Volumes:      make(map[string]struct{}),
			Tty:          true,
			AttachStderr: true,
			AttachStdout: true,
			WorkingDir:   "/",
			Cmd:          []string{"/bin/sh"},

			Hostname:        "",
			Domainname:      "",
			User:            "",
			AttachStdin:     false,
			ExposedPorts:    nat.PortSet{},
			OpenStdin:       false,
			StdinOnce:       false,
			Healthcheck:     nil,
			ArgsEscaped:     false,
			Entrypoint:      []string{},
			NetworkDisabled: false,
			MacAddress:      "",
			OnBuild:         []string{},
			Labels:          map[string]string{},
			StopSignal:      "",
			StopTimeout:     nil,
			Shell:           []string{},
		},
		&container.HostConfig{},
		&network.NetworkingConfig{},
		"",
	)
	if err != nil {
		return DockerError{Original: err}
	}

	ctx.ContainerID = cont.ID

	return nil
}

func (job *Job) attachStd(ctx *jobContext) error {
	_, err := ctx.Client.ContainerAttach(ctx.Context, ctx.ContainerID, types.ContainerAttachOptions{
		Stream: true,
		Stderr: true,
		Stdout: true,
		Logs:   true,

		DetachKeys: "",
		Stdin:      false,
	})
	if err != nil {
		return DockerError{Original: err}
	}

	return nil
}

func (job *Job) execConfig(command []string) types.ExecConfig {
	return types.ExecConfig{
		Cmd:          command,
		AttachStderr: true,
		AttachStdout: true,
		Detach:       false,
		Tty:          false,
		Env:          job.Environment,

		User:        "",
		Privileged:  false,
		AttachStdin: false,
		DetachKeys:  "",
	}
}

func (job *Job) runCommand(ctx *jobContext, command []string) CommandResult {
	result := CommandResult{
		ContainerID: "",
		Command:     command,
		StdOut:      bytes.Buffer{},
		StdErr:      bytes.Buffer{},
		ExitCode:    -1,
		Error:       nil,
	}

	fmt.Printf("[%s] Executing '%s' in '%s'\n", time.Now(), strings.Join(command, " "), job.Image)

	execConfig := job.execConfig(command)

	exec, err := ctx.Client.ContainerExecCreate(ctx.Context, ctx.ContainerID, execConfig)
	if err != nil {
		result.Error = DockerError{Original: err}

		return result
	}

	resp, err := ctx.Client.ContainerExecAttach(ctx.Context, exec.ID, execConfig)
	if err != nil {
		result.Error = DockerError{Original: err}

		return result
	}

	inspect, err := ctx.Client.ContainerExecInspect(ctx.Context, exec.ID)
	if err != nil {
		result.Error = DockerError{Original: err}

		return result
	}

	err = ctx.Client.ContainerExecStart(ctx.Context, exec.ID, types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	})
	if err != nil {
		result.Error = DockerError{Original: err}

		return result
	}

	for !inspect.Running {
		time.Sleep(time.Second)
	}

	_, err = stdcopy.StdCopy(&result.StdOut, &result.StdErr, resp.Reader)
	if err != nil {
		result.Error = err
	}

	result.ExitCode = inspect.ExitCode

	return result
}

func (job *Job) Run(ctx context.Context, client *docker.Client) {
	var err error

	fmt.Printf("Job setup: %s\n", job.Name)

	job.Result = &JobResult{
		Commands: make([]CommandResult, 0),
		Error:    nil,
	}
	jobCtx := &jobContext{
		Client:      client,
		Context:     ctx,
		ContainerID: "",
	}

	err = job.createContainer(jobCtx)
	if err != nil {
		job.Result.Error = err

		return
	}

	err = job.attachStd(jobCtx)
	if err != nil {
		job.Result.Error = err

		return
	}

	err = client.ContainerStart(jobCtx.Context, jobCtx.ContainerID, types.ContainerStartOptions{})
	if err != nil {
		job.Result.Error = DockerError{Original: err}

		return
	}

	for _, command := range job.Commands {
		job.Result.AddCommandResult(job.runCommand(jobCtx, command))
	}

	fmt.Printf("Job teardown: %s\n", job.Name)

	err = client.ContainerStop(ctx, jobCtx.ContainerID, nil)
	if err != nil {
		job.Result.Error = DockerError{Original: err}

		return
	}

	err = client.ContainerRemove(ctx, jobCtx.ContainerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         false,
	})

	if err != nil {
		job.Result.Error = DockerError{Original: err}
	}
}
