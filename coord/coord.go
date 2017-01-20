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
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	pubsub_v1 "google.golang.org/api/pubsub/v1"
)

/*
The coord program listens for certain kinds of messages on a pubsub subscription
(which must already exist), and prints them to stdout so that they can be reviewed
by flargo.
*/

func main() {
	ctx := context.Background()

	if len(os.Args) != 2 {
		fmt.Println("Usage: %s SUBSCRIPTION", os.Args[0])
	}
	subs := os.Args[1]

	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Fatalf("Could not create authed http client: %v", err)
	}

	pubsub, err := pubsub_v1.New(client)
	if err != nil {
		log.Fatalf("Could not create pubsub client: %v", err)
	}

	for {
		resp, err := pubsub.Projects.Subscriptions.Pull(subs, &pubsub_v1.PullRequest{
			MaxMessages: 1,
		}).Do()
		if err != nil {
			log.Printf("Error pulling subscription %q: %v", subs, err)
			continue
		}
		for _, rmsg := range resp.ReceivedMessages {
			// ack rmsg.AckId
			msg := rmsg.Message.Data

			io.Copy(os.Stdout, base64.NewDecoder(base64.StdEncoding, strings.NewReader(msg)))
			fmt.Println()

			if _, err := pubsub.Projects.Subscriptions.Acknowledge(subs, &pubsub_v1.AcknowledgeRequest{
				AckIds: []string{rmsg.AckId},
			}).Do(); err != nil {
				log.Printf("Failed to ack message %q: %v", rmsg.AckId, err)
			}
		}
	}
}
