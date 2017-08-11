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
	"log"
	"os"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

	client := oauth2.NewClient(ctx, google.ComputeTokenSource(""))

	// Poll the subscription until the blocks are resolved.
	pubsub, err := v1pubsub.New(client)
	if err != nil {
		log.Fatalf("Could not create pubsub client: %v", err)
	}

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
			}
		}
	}

	log.Printf("TODO: pull artifacts from %s", gcsPrefix)
}
