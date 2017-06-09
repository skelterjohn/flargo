/*
Copyright 2017 Google Inc. All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	pubsub_v1 "google.golang.org/api/pubsub/v1"
)

/*
The coord program listens for certain kinds of messages on a pubsub subscription
(which must already exist), and prints them to stdout so that they can be reviewed
by flargo.
*/

func init() {
	// Because the metadata package can't figure this out itself.
	os.Setenv("GCE_METADATA_HOST", "metadata.google.internal")
}

func b64Message(msg string) string {
	buf := &bytes.Buffer{}
	io.Copy(base64.NewEncoder(base64.StdEncoding, buf), strings.NewReader(msg))
	return buf.String()
}

func main() {
	ctx := context.Background()

	if len(os.Args) != 3 {
		fmt.Println("Usage: %s WORKFLOW_ID EXECUTION_ID", os.Args[0])
	}

	workflowID := os.Args[1]
	executionID := os.Args[2]
	projectID, err := metadata.ProjectID()
	if err != nil {
		log.Fatalf("Could not get project ID")
	}

	client := oauth2.NewClient(ctx, google.ComputeTokenSource(""))

	pubsub, err := pubsub_v1.New(client)
	if err != nil {
		log.Fatalf("Could not create pubsub client: %v", err)
	}

	tname := fmt.Sprintf("projects/%s/topics/workflow-%s", projectID, workflowID)

	if _, err := pubsub.Projects.Topics.Publish(tname, &pubsub_v1.PublishRequest{
		Messages: []*pubsub_v1.PubsubMessage{{
			Data: b64Message("completed " + executionID),
		}},
	}).Do(); err != nil {
		log.Fatalf("Could not publish message: %v", err)
	}
}
