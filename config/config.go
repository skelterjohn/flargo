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
	"strings"
)

/*
type ':' name '(' name [ 'as' bar ] ')' file
*/

type Config struct {
	Executions []Execution
}

type Execution struct {
	Type   string
	Name   string
	Params []Param
	Path   string
}

type Param struct {
	Name  string
	Alias string
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

			var name, alias string
			name = ptTokens[0]
			if len(ptTokens) > 1 {
				if len(ptTokens) != 3 {
					return nil, fmt.Errorf("line %d: wrong number of tokens for param %q", lineNumber, pt)
				}
				if ptTokens[1] != "as" {
					return nil, fmt.Errorf("line %d: expected 'name as alias'", lineNumber)
				}
				alias = ptTokens[2]
			}
			e.Params = append(e.Params, Param{
				Name:  name,
				Alias: alias,
			})
		}

		e.Path = s

		c.Executions = append(c.Executions, e)
	}

	return &c, nil
}
