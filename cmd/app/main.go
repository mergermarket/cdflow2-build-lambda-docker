package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/mergermarket/cdflow2-build-lambda/internal/app"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "requirements" {
		// requirements is a way for the release container to communicate its requirements to the
		// config container
		if err := json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"needs": []string{"lambda"},
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
	component := os.Getenv("COMPONENT")
	commit := os.Getenv("COMMIT")
	team := os.Getenv("TEAM")
	codeDir := os.Getenv("CDFLOW2_CODE_DIR")

	application := &app.App{}
	params := map[string]interface{}{}
	if err := json.Unmarshal([]byte(os.Getenv("MANIFEST_PARAMS")), &params); err != nil {
		log.Fatalln("error loading MANIFEST_PARAMS:", err)
	}
	if err := application.Run(&app.RunContext{
		Bucket:    bucket,
		BuildID:   buildID,
		CodeDir:   codeDir,
		Version:   version,
		Component: component,
		Commit:    commit,
		Team:      team,
		Params:    params,
	}, os.Stdout, os.Stderr); err != nil {
		log.Fatalln(err)
	}
}
