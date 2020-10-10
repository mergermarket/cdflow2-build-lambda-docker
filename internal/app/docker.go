package app

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/xid"
)

// DockerInterface is a mockable interface for interactions with docker.
type DockerInterface interface {
	RunContainer(codeDir, image string, command []string, outputStream, errorStream io.Writer) error
}

// Docker implements the DockerInterface above to abstract interactions with docker.
type Docker struct {
	Client *client.Client
}

// NewDocker returns an object for interactive with docker.
func NewDocker(client *client.Client) *Docker {
	return &Docker{
		Client: client,
	}
}

// RunContainer runs a docker container.
func (d *Docker) RunContainer(codeDir, image string, command []string, outputStream, errorStream io.Writer) error {
	response, err := d.Client.ContainerCreate(
		context.Background(),
		&container.Config{
			Image:        image,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/code",
			Cmd:          command,
		},
		&container.HostConfig{
			LogConfig: container.LogConfig{Type: "none"},
			Binds:     []string{codeDir + ":/code"},
		},
		nil,
		randomName("cdflow2-release-lambda"),
	)
	if err != nil {
		return err
	}

	hijackedResponse, err := d.Client.ContainerAttach(context.Background(), response.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return err
	}

	if err := d.Client.ContainerStart(
		context.Background(),
		response.ID,
		types.ContainerStartOptions{},
	); err != nil {
		return err
	}

	if _, err := stdcopy.StdCopy(outputStream, errorStream, hijackedResponse.Reader); err != nil {
		return err
	}

	result, err := d.Client.ContainerInspect(context.Background(), response.ID)
	if err != nil {
		return err
	}

	if result.State.Running {
		log.Panicln("unexpected container still running:", result)
	}

	if result.State.ExitCode != 0 {
		return fmt.Errorf("container %s exited with unsuccessful exit code %d", result.ID, result.State.ExitCode)
	}

	return d.Client.ContainerRemove(context.Background(), response.ID, types.ContainerRemoveOptions{})
}

func randomName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, xid.New().String())
}
