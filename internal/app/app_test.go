package app_test

import (
	"log"
	"testing"

	"github.com/docker/docker/client"
	"github.com/mergermarket/cdflow2-release-lambda/internal/app"
)

func TestRun(t *testing.T) {
	// Given
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalln("error intialising docker:", err)
	}
	manifestConfig := map[string]interface{}{}

	// When
	if err := app.Run(dockerClient, "test-bucket", "lambda", "1", manifestConfig); err != nil {
		log.Fatal("error in Run:", err)
	}

	// Then

}
