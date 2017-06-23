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
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type ConfigReport struct {
	Configuration struct {
		ActiveConfiguration string                       `json:"active_configuration"`
		Properties          map[string]map[string]string `json:"properties"`
	} `json:"configuration"`
	Credential struct {
		AccessToken string `json:"access_token"`
		TokenExpiry string `json:"token_expiry"`
	} `json:"credential"`
}

func (cr ConfigReport) GetProperty(section, name string) (string, bool) {
	s, ok := cr.Configuration.Properties[section]
	if !ok {
		return "", false
	}
	p, ok := s[name]
	return p, ok
}

// An SDKConfig provides access to tokens from an account already
// authorized via the Google Cloud SDK.
type SDK struct {
	Account string
}

// NewSDKConfig creates an SDKConfig for the given Google Cloud SDK
// account. If account is empty, the account currently active in
// Google Cloud SDK properties is used.
// Google Cloud SDK credentials must be created by running `gcloud auth`
// before using this function.
// The Google Cloud SDK is available at https://cloud.google.com/sdk/.
func NewSDK(account string) (*SDK, error) {
	return &SDK{
		Account: account,
	}, nil
}

func ReadConfigHelper() (*ConfigReport, error) {
	cmd := exec.Command("gcloud", "config", "config-helper", "--format=json")
	// TODO: use specified account
	//cmd.Env = []string{"CLOUDSDK_CORE_ACCOUNT=" + c.Account}
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting gcloud config: %v", err)
	}

	cfg := &ConfigReport{}
	if err := json.Unmarshal(out, cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling gcloud config: %v", err)
	}

	return cfg, nil
}

func (c *SDK) Token() (*oauth2.Token, error) {
	cfg, err := ReadConfigHelper()
	// eg "2017-06-09T20:04:32Z"
	// rerference time is "Mon Jan 2 15:04:05 -0700 MST 2006"
	et, err := time.Parse("2006-01-02T15:04:05Z", cfg.Credential.TokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("parsing time: %v", err)
	}

	t := &oauth2.Token{
		AccessToken: cfg.Credential.AccessToken,
		Expiry:      et,
	}

	return t, nil
}

// Client returns an HTTP client using Google Cloud SDK credentials to
// authorize requests. The token will acquired from
// `gcloud config config-helper` every time, and `gcloud` will be
// responsible for refreshing it as needed.
func (c *SDK) Client(ctx context.Context) *http.Client {
	return &http.Client{
		Transport: &oauth2.Transport{
			Source: c,
		},
	}
}
