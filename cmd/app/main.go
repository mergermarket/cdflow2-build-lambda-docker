package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/docker/docker/client"
	"github.com/mergermarket/cdflow2-release-lambda/internal/app"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "requirements" {
		// requirements is a way for the release container to communciate its requirements to the
		// config container
		if err := json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"env": []string{"LAMBDA_BUCKET", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION"},
		}); err != nil {
			log.Panicln("error encoding requirements:", err)
		}
		return
	}
	// declared above
	bucket := os.Getenv("LAMBDA_BUCKET")

	// built-in
	buildID := os.Getenv("BUILD_ID")
	version := os.Getenv("VERSION")

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalln("error intialising docker:", err)
	}

	params := map[string]interface{}{}
	if err := json.Unmarshal([]byte(os.Getenv("MANIFEST_PARAMS")), &params); err != nil {
		log.Fatalln("error loading MANIFEST_PARAMS:", err)
	}
	if err := app.Run(dockerClient, bucket, buildID, version, params); err != nil {
		log.Fatalln(err)
	}
}
