// Copyright 2011 Arne Roomann-Kurrik.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oauth1a

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func LoadCredentials() (string, string, string, string, error) {
	credentials, err := ioutil.ReadFile("CREDENTIALS")
	if err != nil {
		return "", "", "", "", err
	}
	lines := strings.Split(string(credentials), "\n")
	return lines[0], lines[1], lines[2], lines[3], nil
}

func GetTwitterConfig(t *testing.T) (*Service, *UserConfig) {
	key, secret, token, token_secret, err := LoadCredentials()
	if err != nil {
		t.Fatal("Could not load CREDENTIALS file")
	}
	service := &Service{
		RequestURL:   "https://api.twitter.com/oauth/request_token",
		AuthorizeURL: "https://api.twitter.com/oauth/authorize",
		AccessURL:    "https://api.twitter.com/oauth/access_token",
		ClientConfig: &ClientConfig{
			ConsumerKey:    key,
			ConsumerSecret: secret,
			CallbackURL:    "oob",
		},
		Signer: new(HmacSha1Signer),
	}
	user := NewAuthorizedConfig(token, token_secret)
	return service, user
}

func TestIntegration(t *testing.T) {
	if testing.Short() == true {
		t.Log("Not running integration test because short was specified.")
		return
	}
	service, userConfig := GetTwitterConfig(t)
	httpClient := new(http.Client)
	url := "https://api.twitter.com/1.1/account/verify_credentials.json"
	httpRequest, _ := http.NewRequest("GET", url, nil)
	service.Sign(httpRequest, userConfig)
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		t.Error("Response had an error")
	}
	if httpResponse.StatusCode != 200 {
		t.Errorf("Response returned code of %v", httpResponse.StatusCode)
	}
	if body, err := ioutil.ReadAll(httpResponse.Body); err == nil {
		t.Logf("Got %v\n", string(body))
	}
}

func TestGetRequestToken(t *testing.T) {
	if testing.Short() == true {
		t.Log("Not running integration test because short was specified.")
		return
	}
	service, _ := GetTwitterConfig(t)
	userConfig := new(UserConfig)
	httpClient := new(http.Client)
	err := userConfig.GetRequestToken(service, httpClient)
	if err != nil {
		t.Errorf("Response had an error: %v", err)
	}
}
