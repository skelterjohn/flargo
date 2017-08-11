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
	"errors"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	v1cloudbuild "google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	v1pubsub "google.golang.org/api/pubsub/v1"

	"github.com/skelterjohn/flargo/auth"
	"github.com/skelterjohn/flargo/config"
	"github.com/skelterjohn/flargo/executions"
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

// TODO: configure gcr.io/cloud-workflows/<flargo image> to use customer project registries.

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
		cfg, err := config.Load(cfgFile)
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

func start(ctx context.Context, cfg *config.Config) error {
	scfg, err := auth.NewSDK("")
	if err != nil {
		return fmt.Errorf("could not find SDK config: %v", err)
	}

	ch, err := auth.ReadConfigHelper()
	if err != nil {
		return fmt.Errorf("could not read SDK config helper: %v", err)
	}

	projectID, ok := ch.GetProperty("core", "project")
	if !ok {
		return errors.New("no project property set")
	}

	cb, err := v1cloudbuild.New(scfg.Client(ctx))
	if err != nil {
		return fmt.Errorf("could not create cloudbuild client: %v", err)
	}
	ps, err := v1pubsub.New(scfg.Client(ctx))
	if err != nil {
		return fmt.Errorf("could not create pubsub client: %v", err)
	}
	sc, err := storage.NewClient(ctx, option.WithTokenSource(scfg))
	if err != nil {
		return fmt.Errorf("could not create storage client: %v", err)
	}
	executionsClient := executions.Client{
		ProjectID: projectID,
		Builds:    cb,
		Storage:   sc,
	}

	cfgDir, _ := filepath.Split(cfg.Path)

	// Load execution configs
	bconfigs := map[string]*v1cloudbuild.Build{}
	for _, execution := range cfg.Executions {
		b, err := executions.LoadBuild(filepath.Join(cfgDir, execution.Path))
		if err != nil {
			return err
		}
		bconfigs[execution.Name] = b
	}

	// Start coord
	op, err := cb.Projects.Builds.Create(projectID, &v1cloudbuild.Build{
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

	workflowID := b.Id

	log.Printf("Workflow ID: %s", workflowID)

	for {
		execLog, err := executionsClient.FetchBuildLog(ctx, workflowID)
		if err != nil && err != storage.ErrObjectNotExist {
			return fmt.Errorf("could not fetch workflow log: %v", err)
		}
		if err == nil && strings.Contains(execLog, "Created topic") {
			break
		}
		time.Sleep(time.Second)
	}
	// This topic was created by the coord execution.
	tname := fmt.Sprintf("workflow-%s", workflowID)
	workflowTopic := fmt.Sprintf("projects/%s/topics/%s", projectID, tname)
	log.Printf("worklow topic: %s", workflowTopic)

	// Ensure a GCS place for artifacts.
	gcsBucket := fmt.Sprintf("%s_workflow_artifacts", projectID)
	gcsPrefix := fmt.Sprintf("gs://%s/%s", gcsBucket, workflowID)
	// Create the bucket. If it exists, ensure that it's owned by this project to avoid artifact theft.
	if err := sc.Bucket(gcsBucket).Create(ctx, projectID, nil); err != nil {
		// if 409, fetch the bucket to compare project IDs.
		gerr, ok := err.(*googleapi.Error)
		if ok && gerr.Code == 409 {
			policy, err := sc.Bucket(gcsBucket).IAM().Policy(ctx)
			if err != nil {
				return fmt.Errorf("could not check policy of gs://%s: %v", gcsBucket, err)
			}
			if !policy.HasRole("projectOwner:"+projectID, "roles/storage.legacyBucketOwner") {
				jdata, _ := json.MarshalIndent(policy, " ", " ")
				log.Printf("Artifacts bucket policy:\n%s\n", jdata)
				return errors.New("artifacts bucket exists, but is owned by someone else")
			}
		} else {
			return fmt.Errorf("could not create artifact bucket: %v", err)
		}
	}

	// For each execution,
	execErrors := make(chan error)
	var execWG sync.WaitGroup
	for i, execution := range cfg.Executions {
		execWG.Add(1)
		go func(i int, execution config.Execution) {
			defer execWG.Done()
			// - Create subscription
			sname := fmt.Sprintf("workflow-%s-%d", workflowID, i)
			executionSubscription := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, sname)

			log.Printf("%q execution subscription: %s", execution.Name, sname)
			if _, err := ps.Projects.Subscriptions.Create(executionSubscription, &v1pubsub.Subscription{
				Name:  sname,
				Topic: workflowTopic,
			}).Do(); err != nil {
				execErrors <- fmt.Errorf("could not create %q subscription: %v", execution.Name, err)
				return
			}

			var waitExecutions []string
			for _, param := range execution.Params {
				waitExecutions = append(waitExecutions, param.Name)
			}

			// - Augment steps with wait/complete
			build := bconfigs[execution.Name]
			build.Steps = append([]*v1cloudbuild.BuildStep{{
				Name: "gcr.io/cloud-workflows/wait",
				Args: append(
					[]string{
						gcsPrefix,
						workflowID,
						executionSubscription,
					},
					waitExecutions...,
				),
			}}, build.Steps...)
			build.Steps = append(build.Steps,
				&v1cloudbuild.BuildStep{
					Name: "gcr.io/cloud-workflows/complete",
					Args: []string{
						"gs://todo",
						workflowID,
						execution.Name,
					},
				},
			)

			// - Begin execution
			op, err := cb.Projects.Builds.Create(projectID, build).Context(ctx).Do()
			if err != nil {
				execErrors <- fmt.Errorf("could not create %q execution: %v", execution.Name, err)
				return
			}
			if executionBuild, err := buildFromOp(op); err != nil {
				execErrors <- fmt.Errorf("could not understand %q execution: %v", execution.Name, err)
				return
			} else {
				log.Printf("%q execution is build %s", execution.Name, executionBuild.Id)
			}
		}(i, execution)
	}

	execWG.Wait()
	close(execErrors)
	for err := range execErrors {
		return err
	}

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
