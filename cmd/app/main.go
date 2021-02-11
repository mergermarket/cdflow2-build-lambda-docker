package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	path := os.Getenv("LAMBDA_PATH")

	// built-in
	buildID := os.Getenv("BUILD_ID")
	codeDir := os.Getenv("CDFLOW2_CODE_DIR")

	application := &app.App{}
	params := map[string]interface{}{}
	if err := json.Unmarshal([]byte(os.Getenv("MANIFEST_PARAMS")), &params); err != nil {
		log.Fatalln("error loading MANIFEST_PARAMS:", err)
	}
	metadata, err := application.Run(&app.RunContext{
		Bucket:  bucket,
		BuildID: buildID,
		CodeDir: codeDir,
		Path:    path,
		Params:  params,
	}, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile("/release-metadata.json", data, 0644); err != nil {
		log.Fatalln("error writing release metadata:", err)
	}
}
