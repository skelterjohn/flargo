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
package executions

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	v1cloudbuild "google.golang.org/api/cloudbuild/v1"
	"gopkg.in/yaml.v2"
)

func LoadBuild(path string) (*v1cloudbuild.Build, error) {
	efile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config %q: %v", path, err)
	}
	edata, err := ioutil.ReadAll(efile)
	if err != nil {
		return nil, fmt.Errorf("could not read config %q: %v", path, err)
	}
	b := &v1cloudbuild.Build{}
	if err := yaml.Unmarshal(edata, b); err != nil {
		return nil, fmt.Errorf("could not parse config %q: %v", path, err)
	}
	return b, nil
}

type Client struct {
	ProjectID string
	Builds    *v1cloudbuild.Service
	Storage   *storage.Client
}

func (c Client) WaitForBuild(ctx context.Context, buildID string) error {
	return nil
}

func (c Client) FetchBuildStatus(ctx context.Context, buildID string) (string, error) {
	return "", nil
}

func (c Client) FetchBuildLog(ctx context.Context, buildID string) (string, error) {
	b, err := c.Builds.Projects.Builds.Get(c.ProjectID, buildID).Do()
	if err != nil {
		return "", err
	}
	logfilePath := fmt.Sprintf("%s/log-%s.txt", b.LogsBucket, b.Id)
	tokens := strings.SplitN(logfilePath[len("gs://"):], "/", 2)
	bucket := tokens[0]
	object := tokens[1]

	r, err := c.Storage.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return "", err
	}
	d, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	return string(d), nil
}
