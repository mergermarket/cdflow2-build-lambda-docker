package app

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/docker/docker/client"
)

// App is the application we are running.
type App struct {
	dockerClient *client.Client
	S3Client     s3iface.S3API
}

func (app *App) getDockerClient() *client.Client {
	if app.dockerClient == nil {
		dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			log.Panic("error connecting to docker:", err)
		}
		app.dockerClient = dockerClient
	}
	return app.dockerClient
}

func (app *App) getS3Client() s3iface.S3API {
	if app.S3Client == nil {
		app.S3Client = s3.New(session.Must(session.NewSession(&aws.Config{})))
	}
	return app.S3Client
}

// RunContext contains the context that the build container is run in.
type RunContext struct {
	Docker        DockerInterface
	Bucket        string
	Path          string
	BuildID       string
	CodeDir       string
	MappedCodeDir string
	Params        map[string]interface{}
}

// Run runs the release.
func (app *App) Run(context *RunContext, outputStream, errorStream io.Writer) (map[string]string, error) {
	config, err := getConfig(context.BuildID, context.Params)
	if err != nil {
		return nil, fmt.Errorf("error getting config: %w", err)
	}

	docker := context.Docker
	if docker == nil {
		docker = NewDocker(app.getDockerClient())
	}

	fmt.Fprintf(errorStream, "\ncdflow2-build-lambda: running %q in %q\n", config.command, config.image)

	if err := docker.RunContainer(context.CodeDir, config.image, config.command, outputStream, errorStream); err != nil {
		return nil, fmt.Errorf("error running container: %w", err)
	}

	tmpfile, err := ioutil.TempFile("", "cdflow2-release-lambda-*")
	if err != nil {
		return nil, fmt.Errorf("error getting tempfile: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	mappedCodeDir := context.MappedCodeDir
	if mappedCodeDir == "" {
		mappedCodeDir = "/code"
	}

	target := path.Join(mappedCodeDir, config.target)
	targetInfo, err := os.Stat(target)

	fmt.Fprintf(os.Stderr, "\ncdflow2-build-lambda: zipping target %q\n\n", config.target)

	if err != nil {
		return nil, fmt.Errorf("target '%s' does not exist: %w", config.target, err)
	}
	if targetInfo.IsDir() {
		if err := zipDir(tmpfile, target); err != nil {
			return nil, fmt.Errorf("error zipping directory: %w", err)
		}
	} else {
		if err := zipFile(tmpfile, target); err != nil {
			return nil, fmt.Errorf("error zipping file: %w", err)
		}
	}
	if err := tmpfile.Sync(); err != nil {
		return nil, fmt.Errorf("error syncing write on zipfile: %w", err)
	}
	if _, err := tmpfile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("error seeking zipfile: %w", err)
	}

	bucket := context.Bucket
	key := context.Path

	fmt.Fprintf(os.Stderr, "\ncdflow2-build-lambda: uploading zip to s3://%s/%s...", bucket, key)

	s3client := app.getS3Client()
	if _, err := s3client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   tmpfile,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "\n\n")
		return nil, fmt.Errorf("error uploading to s3: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\ndone.\n\n")

	return map[string]string{
		"bucket": bucket,
		"key":    key,
	}, nil
}

type config struct {
	image   string
	target  string
	handler string
	command []string
}

func getConfig(buildID string, params map[string]interface{}) (*config, error) {
	result := config{}
	var ok bool
	if result.image, ok = params["image"].(string); !ok {
		return nil, fmt.Errorf("unexpected type for build.%v.params.image: %T (should be string)", buildID, params["image"])
	}
	if result.target, ok = params["target"].(string); !ok {
		return nil, fmt.Errorf("unexpected type for build.%v.params.target: %T (should be string)", buildID, params["target"])
	}
	if result.handler, ok = params["handler"].(string); !ok {
		return nil, fmt.Errorf("unexpected type for build.%v.params.handler: %T (should be string)", buildID, params["handler"])
	}
	if command, ok := params["command"].(string); ok {
		result.command = []string{"/bin/sh", "-e", "-c", command}
	} else if command, ok := params["command"].([]string); ok {
		result.command = command
	} else {
		return nil, fmt.Errorf("unexpected type for build.%v.params.command: %T (should be string or array of strings)", buildID, params["command"])
	}
	return &result, nil
}

func zipFile(writer io.Writer, file string) error {
	zipWriter := zip.NewWriter(writer)
	name := filepath.Base(file)
	writer, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	reader, err := os.Open(file)
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(writer, reader)
	if err != nil {
		return err
	}
	return zipWriter.Close()
}

func zipDir(writer io.Writer, dir string) error {
	zipWriter := zip.NewWriter(writer)
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relativePath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		writer, err := zipWriter.Create(relativePath)
		if err != nil {
			return err
		}

		reader, err := os.Open(path)
		if err != nil {
			return err
		}
		defer reader.Close()

		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return zipWriter.Close()
}
