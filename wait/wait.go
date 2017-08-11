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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	v1pubsub "google.golang.org/api/pubsub/v1"
)

type completionMessage struct {
	Completed string `json:"completed"`
	Artifacts string `json:"artifacts"`
}

func usage() {
	log.Fatalf("Usage: wait GCS_PREFIX WORKFLOW_ID SUBSCRIPTION BLOCKING_EXECUTION*")
}

func main() {
	ctx := context.Background()

	args := os.Args[1:]
	if len(args) < 3 {
		usage()
	}
	gcsPrefix := args[0]
	//workflowID := args[1]
	subscriptionName := args[2]
	blocks := map[string]bool{}
	for _, block := range args[3:] {
		blocks[block] = true
	}

	if !strings.HasPrefix(gcsPrefix, "gs://") {
		log.Fatalf("Invalid GCS prefix %q", gcsPrefix)
	}
	bucketObject := gcsPrefix[len("gs://"):]
	tokens := strings.SplitN(bucketObject, "/", 2)
	if len(tokens) != 2 {
		log.Fatalf("Invalid GCS prefix %q", gcsPrefix)
	}
	bucket, object := tokens[0], tokens[1]

	client := oauth2.NewClient(ctx, google.ComputeTokenSource(""))

	pubsub, err := v1pubsub.New(client)
	if err != nil {
		log.Fatalf("Could not create pubsub client: %v", err)
	}

	sc, err := storage.NewClient(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Could not create storage client: %v", err)
	}

	// Poll the subscription until the blocks are resolved.
	for len(blocks) > 0 {
		resp, err := pubsub.Projects.Subscriptions.Pull(subscriptionName, &v1pubsub.PullRequest{
			MaxMessages: 10,
		}).Do()
		if err != nil {
			log.Fatalf("Could not pull from subscription: %v", err)
		}
		for _, rmsg := range resp.ReceivedMessages {
			if _, err := pubsub.Projects.Subscriptions.Acknowledge(subscriptionName, &v1pubsub.AcknowledgeRequest{
				AckIds: []string{rmsg.AckId},
			}).Do(); err != nil {
				log.Printf("Could not ack %q: %v", rmsg.AckId, err)
			}

			dec := json.NewDecoder(base64.NewDecoder(base64.StdEncoding, strings.NewReader(rmsg.Message.Data)))
			var cmsg completionMessage
			if err := dec.Decode(&cmsg); err != nil {
				log.Printf("Could not decode message: %v", err)
			}
			if cmsg.Completed != "" && blocks[cmsg.Completed] {
				log.Printf("Got completion %+q", cmsg)
				delete(blocks, cmsg.Completed)

				// copy the blocking execution's artifacts into this execution.
				if err := fetchArtifacts(ctx, sc, bucket, object, cmsg.Completed); err != nil {
					log.Fatalf("Could not fetch artifacts for %q: %v", cmsg.Completed, err)
				}
			}
		}
	}

	if err := os.MkdirAll(filepath.Join("/workflow_artifacts", "out"), 0755); err != nil {
		log.Fatal("could not make artifact out directory")
	}
}

func fetchArtifacts(ctx context.Context, sc *storage.Client, bucket, object, block string) error {
	objItr := sc.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix: path.Join(object, block),
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	for {
		attrs, err := objItr.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		objName := attrs.Name
		wg.Add(1)
		go func(objName string) {
			defer wg.Done()
			r, err := sc.Bucket(bucket).Object(objName).NewReader(ctx)
			if err != nil {
				errCh <- fmt.Errorf("could not read object %q: %v", objName, err)
				return
			}

			relpath, err := filepath.Rel(object, objName)
			if err != nil {
				errCh <- fmt.Errorf("could not get relative path from %q to %q", object, objName)
			}

			localPath := filepath.Join("/workflow_artifacts", "in", relpath)
			localDir, _ := filepath.Split(localPath)
			if err := os.MkdirAll(localDir, 0755); err != nil {
				errCh <- fmt.Errorf("could not create directory for artifact %q: %v", localDir, err)
				return
			}

			fout, err := os.Create(localPath)
			if err != nil {
				errCh <- fmt.Errorf("could not create local artifact %q: %v", relpath, err)
				return
			}

			if _, err := io.Copy(fout, r); err != nil {
				errCh <- fmt.Errorf("could not download artifact %q: %v", relpath, err)
				return
			}
			log.Printf("Downloaded %s to %s", objName, fout.Name())
		}(objName)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		return err
	}

	return nil
}
