package app

import "github.com/docker/docker/client"

// Run runs the release.
func Run(client *client.Client, bucket, buildID, version string, params map[string]interface{}) error {
	return nil
}
