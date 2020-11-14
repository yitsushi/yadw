# Yet Another Docker Workflow

**WIP**, bookmark this repo and come back later.

### Example:

```go
package main

import (
	"context"
	"log"

	"github.com/docker/docker/client"
	"github.com/yitsushi/yadw/workflow"
)

func main() {
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Println(err)
	}

	wfl := workflow.NewWorkflow(cli)
	wfl.AddJob(&workflow.Job{
		Name:  "Check",
		Image: "alpine:latest",
		Commands: [][]string{
			{"touch", "/asdasd"},
			{"ls", "-la"},
		},
		Environment: []string{},
		StopOnError: true,
		Result:      nil,
	})

	wfl.Execute(context.Background())

	for _, job := range wfl.Jobs {
		if job.Result.Error != nil {
			log.Printf("Job level error on %s: %s", job.Name, job.Result.Error.Error())
		}

		for _, command := range job.Result.Commands {
			if job.Result.Error != nil {
				log.Printf("Command level error on %s: %s", command.Command, command.Error.Error())
			}

			log.Printf("Stdout:\n%s", command.StdOut.String())
		}
	}
}
```
