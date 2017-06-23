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
package main // import "github.com/skelterjohn/flargo"

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/net/context"
	v1cloudbuild "google.golang.org/api/cloudbuild/v1"

	"github.com/skelterjohn/flargo/auth"
	"github.com/skelterjohn/flargo/config"
)

func usage() {
	log.Fatal(`flargo is a tool to run workflows on top of Google Container Engine.

Usage: flargo start CONFIG
              wait FLOW
              describe FLOW
              retry FLOW EXECUTION
              skip FLOW EXECUTION
`)
}

func main() {
	ctx := context.Background()

	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		usage()
	}
	switch args[0] {
	case "start":
		if len(args) != 2 {
			usage()
		}
		cfgFile := args[1]
		fin, err := os.Open(cfgFile)
		if err != nil {
			log.Fatalf("Could not open %q: %v", cfgFile, err)
		}
		cfg, err := config.Parse(fin)
		if err != nil {
			log.Fatalf("Could not parse %q: %v", cfgFile, err)
		}
		if err := start(ctx, cfg); err != nil {
			log.Fatalf("Could not start workflow: %v", err)
		}
	}

	// WAIT PROCESS
	// Create subscription to watch for completions
	// Inspect coord logs for completions
	// Wait for remaining completions on pubsub

	// DESCRIBE PROCESS
	// ?

	// RETRY PROCESS
	// Cancel existing execution
	// Begin execution

	// SKIP PROCESS
	// Cancel existing execution
	// Publish completion
}

func buildFromOp(op *v1cloudbuild.Operation) (*v1cloudbuild.Build, error) {
	md := struct {
		T     string              `json:"@type"`
		Build *v1cloudbuild.Build `json:"build"`
	}{}
	if err := json.Unmarshal(op.Metadata, &md); err != nil {
		return nil, err
	}
	return md.Build, nil
}

func getCloudbuildClient(ctx context.Context) (*v1cloudbuild.Service, error) {
	scfg, err := auth.NewSDK("")
	if err != nil {
		return nil, err
	}
	return v1cloudbuild.New(scfg.Client(ctx))
}

func start(ctx context.Context, cfg *config.Config) error {
	cb, err := getCloudbuildClient(ctx)
	if err != nil {
		return fmt.Errorf("could not create cloudbuild client: %v", err)
	}

	// Start coord
	op, err := cb.Projects.Builds.Create("cloud-workflows", &v1cloudbuild.Build{
		Steps: []*v1cloudbuild.BuildStep{{
			Name: "gcr.io/cloud-workflows/coord",
			Args: []string{"$BUILD_ID"},
		}},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("could not create coord execution: %v", err)
	}

	b, err := buildFromOp(op)
	if err != nil {
		return fmt.Errorf("could not unmarshal build: %v", err)
	}
	log.Println(b.Id)

	// Create subscriptions for each of the executions
	// Begin executions

	return nil
}

type coord struct {
}

func startCoord() (*coord, error) {
	return nil, nil
}

func getCoord(workflowID string) (*coord, error) {
	return nil, nil
}
