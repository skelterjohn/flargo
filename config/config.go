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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

/*
type ':' name '(' name [ 'as' bar ] ')' file
*/

type Config struct {
	Executions []Execution
	Path       string
}

type Execution struct {
	Type   string
	Name   string
	Params []Param
	Path   string
}

type Param struct {
	Name string
}

func Load(path string) (*Config, error) {
	fin, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %q: %v", path, err)
	}
	cfg, err := Parse(fin)
	if err != nil {
		return nil, fmt.Errorf("could not parse %q: %v", path, err)
	}

	cfg.Path = path

	return cfg, nil
}

func Parse(r io.Reader) (*Config, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	var c Config

	for lineNumber, l := range lines {
		s := strings.TrimSpace(l)
		if s == "" || s[0] == '#' {
			continue
		}
		var e Execution
		colonStop := strings.Index(s, ":")
		if colonStop == -1 {
			return nil, fmt.Errorf("line %d: expected '^<type> :'", lineNumber)
		}
		e.Type = strings.TrimSpace(s[:colonStop])
		s = strings.TrimSpace(s[colonStop+1:])
		parenStop := strings.Index(s, "(")
		if parenStop == -1 {
			return nil, fmt.Errorf("line %d: expected 'name ('", lineNumber)
		}
		e.Name = strings.TrimSpace(s[:parenStop])
		s = strings.TrimSpace(s[parenStop+1:])

		parenStop = strings.Index(s, ")")
		if parenStop == -1 {
			return nil, fmt.Errorf("line %d: expected '( param, param, ... )'", lineNumber)
		}
		ps := s[:parenStop]
		s = strings.TrimSpace(s[parenStop+1:])
		paramTokens := strings.Split(ps, ",")
		for _, pt := range paramTokens {
			if pt == "" {
				break
			}
			ptTokens := strings.Fields(pt)

			var name string
			name = ptTokens[0]
			if len(ptTokens) != 1 {
				return nil, fmt.Errorf("line %d: wrong number of tokens for param %q", lineNumber, pt)
			}
			e.Params = append(e.Params, Param{
				Name: name,
			})
		}

		e.Path = s

		c.Executions = append(c.Executions, e)
	}

	names := map[string]bool{}
	for _, e := range c.Executions {
		if names[e.Name] {
			return nil, fmt.Errorf("repeated name %q", e.Name)
		}
		names[e.Name] = true
	}

	return &c, nil
}
