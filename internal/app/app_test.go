package app_test

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/mergermarket/cdflow2-build-lambda/internal/app"
)

type mockedDocker struct{}

func (d *mockedDocker) RunContainer(codeDir, image string, command []string, outputStream, errorStream io.Writer) error {
	outputStream.Write([]byte("test output\n"))
	errorStream.Write([]byte("test error output\n"))
	return Copy(path.Join(codeDir, "test.txt"), path.Join(codeDir, "app"))
}

// Copy source to destination.
func Copy(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

type mockedS3 struct {
	s3iface.S3API
	contents map[string]string
	uploaded bytes.Buffer
}

func (m *mockedS3) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	data, err := ioutil.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	m.contents = make(map[string]string)
	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		d, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		m.contents[f.Name] = string(d)
	}
	return &s3.PutObjectOutput{}, nil
}

func getCodeDir() string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("could not get filename for code dir")
	}
	return path.Join(path.Dir(filename), "../../test/code")
}

func TestRun(t *testing.T) {
	// Given
	manifestConfig := map[string]interface{}{
		"image":   "alpine",
		"target":  "app",
		"handler": "app",
		"command": "echo test output; echo test error output >&2; cp test.txt app",
	}
	s3Client := &mockedS3{}
	application := &app.App{
		S3Client: s3Client,
	}
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer
	codeDir := getCodeDir()

	var docker app.DockerInterface = nil
	if os.Getenv("TEST_NO_DOCKER") == "true" {
		docker = &mockedDocker{}
	}

	// When
	if err := application.Run(&app.RunContext{
		Docker:    docker,
		Bucket:    "test-bucket",
		BuildID:   "lambda",
		CodeDir:   codeDir,
		Version:   "1",
		Component: "test-component",
		Commit:    "test-commit",
		Team:      "test-team",
		Params:    manifestConfig,
	}, &outputBuffer, &errorBuffer); err != nil {
		t.Fatalf("error in Run: %s\n  output: %q", err, errorBuffer.String())
	}

	// Then
	if !strings.Contains(outputBuffer.String(), "test output\n") {
		t.Fatalf("missing output in %#v", outputBuffer.String())
	}
	if !strings.Contains(errorBuffer.String(), "test error output\n") {
		t.Fatalf("missing error output in %#v", errorBuffer.String())
	}
	expected := map[string]string{"app": "test content"}
	if !reflect.DeepEqual(s3Client.contents, expected) {
		t.Fatalf("got %#v, expected %#v", s3Client.contents, expected)
	}
}
