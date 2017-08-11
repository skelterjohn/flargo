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
package config

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestBasicParse(t *testing.T) {
	r := strings.NewReader(`
# build will begin immediately.
exec: build() build.yaml

# build will begin immediately.
exec: build_probes() probe.yaml

# "deploy_to_dev" will only start once "build" is complete. Its "out" files will
# be put in "in/build".
# This example assumpes that retagging an image is sufficient to deploy. That
# is, it assumes that the runtimes are watching for pushes to specific tags.
exec: deploy_to_dev(build) deploy_dev.yaml

# Once dev is ready, run the probes (built earlier) to check that things are
# working.
exec: test_dev(deploy_to_dev, build_probes) test_dev.yaml

# A wait directive means that a human has to run flargo to indicate this
# execution has completed. It's used as a manual gate between dev and prod,
# in this example.
wait: dev_to_prod(test_dev) -

# "deploy_to_prod" will only start once "dev_to_prod" is complete.
exec: deploy_to_prod(dev_to_prod) deploy_prod.yaml


# Once prod is ready, run the probes (built earlier) to check that things are
# working.
exec: test_prod(deploy_to_prod, build_probes) test_prod.yaml
`)
	expected := Config{
		Executions: []Execution{{
			Type: "exec",
			Name: "build",
			Path: "build.yaml",
		}, {
			Type: "exec",
			Name: "build_probes",
			Path: "probe.yaml",
		}, {
			Type: "exec",
			Name: "deploy_to_dev",
			Params: []Param{{
				Name: "build",
			}},
			Path: "deploy_dev.yaml",
		}, {
			Type: "exec",
			Name: "test_dev",
			Params: []Param{{
				Name: "deploy_to_dev",
			}, {
				Name: "build_probes",
			}},
			Path: "test_dev.yaml",
		}, {
			Type: "wait",
			Name: "dev_to_prod",
			Params: []Param{{
				Name: "test_dev",
			}},
			Path: "-",
		}, {
			Type: "exec",
			Name: "deploy_to_prod",
			Params: []Param{{
				Name: "dev_to_prod",
			}},
			Path: "deploy_prod.yaml",
		}, {
			Type: "exec",
			Name: "test_prod",
			Params: []Param{{
				Name: "deploy_to_prod",
			}, {
				Name: "build_probes",
			}},
			Path: "test_prod.yaml",
		}},
	}
	config, err := Parse(r)
	if err != nil {
		t.Fatal(err)
	}
	jdata, _ := json.MarshalIndent(config, " ", " ")
	t.Logf("%s\n", jdata)
	if len(config.Executions) != len(expected.Executions) {
		t.Error("wrong")
	} else {
		for i := range expected.Executions {
			if !reflect.DeepEqual(config.Executions[i], expected.Executions[i]) {
				t.Errorf("item %d: got\n%+v\nwant:\n%+v\n", i, config.Executions[i], expected.Executions[i])
			}
		}
	}
}
