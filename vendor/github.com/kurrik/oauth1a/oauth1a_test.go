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
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

var user = NewAuthorizedConfig("token", "secret")

var client = &ClientConfig{
	ConsumerKey:    "consumer_key",
	ConsumerSecret: "consumer_secret",
	CallbackURL:    "https://example.com/callback",
}

var signer = new(HmacSha1Signer)

var service = &Service{
	RequestURL:   "https://example.com/request_token",
	AuthorizeURL: "https://example.com/request_token",
	AccessURL:    "https://example.com/request_token",
	ClientConfig: client,
	Signer:       signer,
}

func TestSignature(t *testing.T) {
	api_url := "https://example.com/endpoint"
	request, _ := http.NewRequest("GET", api_url, nil)
	service.Sign(request, user)
	params, _ := signer.GetOAuthParams(request, client, user, "nonce", "timestamp")
	signature := params["oauth_signature"]
	expected := "8+ZC6DP8FU3z50qSWDeYCGix2x0="
	if signature != expected {
		t.Errorf("Signature %v did not match expected %v", signature, expected)
	}
}

func TestNewlineInParameter(t *testing.T) {
	api_url := "https://api.twitter.com/1.1/statuses/update.json"
	data := url.Values{}
	data.Set("status", "Hello\nWorld")
	body := strings.NewReader(data.Encode())
	request, _ := http.NewRequest("POST", api_url, body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	service.Sign(request, user)
	nonce := "a87d3d52a22f467e956bd62aece16386"
	timestamp := "1364836266"
	params, base := signer.GetOAuthParams(request, client, user, nonce, timestamp)
	t.Logf("Signature Base String: %v\n", base)
	signature := params["oauth_signature"]
	expected := "5ODYPjb2rDgLfTEiJBdtaeqx0tw="
	if signature != expected {
		t.Errorf("Signature %v did not match expected %v", signature, expected)
	}
}

func TestNewLineCarriageReturnInParameter(t *testing.T) {
	api_url := "https://api.twitter.com/1.1/statuses/update.json"
	data := url.Values{}
	data.Set("status", "Hello\r\nWorld")
	body := strings.NewReader(data.Encode())
	request, _ := http.NewRequest("POST", api_url, body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	service.Sign(request, user)
	nonce := "a87d3d52a22f467e956bd62aece16386"
	timestamp := "1364836266"
	params, base := signer.GetOAuthParams(request, client, user, nonce, timestamp)
	t.Logf("Signature Base String: %v\n", base)
	signature := params["oauth_signature"]
	expected := "b51+W8Igb0m4xcP1Hysr6CBJo4o="
	if signature != expected {
		t.Errorf("Signature %v did not match expected %v", signature, expected)
	}
}

func TestSlashInParameter(t *testing.T) {
	api_url := "https://stream.twitter.com/1.1/statuses/filter.json"
	data := url.Values{}
	data.Set("track", "example.com/abcd")
	body := strings.NewReader(data.Encode())
	request, _ := http.NewRequest("POST", api_url, body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	service.Sign(request, user)
	nonce := "bf2cb6d611e59f99103238fc9a3bb8d8"
	timestamp := "1362434376"
	params, _ := signer.GetOAuthParams(request, client, user, nonce, timestamp)
	signature := params["oauth_signature"]
	expected := "LcxylEOnNdgoKSJi7jX07mxcvfM="
	if signature != expected {
		t.Errorf("Signature %v did not match expected %v", signature, expected)
	}
}

func TestSlashInQuerystring(t *testing.T) {
	var (
		expected  string
		raw_query string
		api_url   string
		body      io.Reader
		request   *http.Request
		nonce     string = "884275759fbab914654b50ae643c563a"
		timestamp string = "1362435218"
	)
	api_url = "https://stream.twitter.com/1.1/statuses/filter.json?track=example.com/query"
	request, _ = http.NewRequest("POST", api_url, body)
	service.Sign(request, user)

	expected = "track=example.com%2Fquery"
	raw_query = request.URL.RawQuery
	if raw_query != expected {
		t.Errorf("Query parameter incorrect, got %v, expected %v", raw_query, expected)
	}
	params, _ := signer.GetOAuthParams(request, client, user, nonce, timestamp)
	signature := params["oauth_signature"]
	expected = "OAldqvRrKDXRGZ9BqSi2CqeVH0g="
	if signature != expected {
		t.Errorf("Signature %v did not match expected %v", signature, expected)
	}
}

func TestMultipleQueryValues(t *testing.T) {
	var (
		expected  string
		raw_query string
		api_url   string
		body      io.Reader
		request   *http.Request
	)
	api_url = "https://stream.twitter.com/1.1/statuses/filter.json?track=example&count=200&count=100"
	request, _ = http.NewRequest("POST", api_url, body)
	service.Sign(request, user)

	expected = "count=100&count=200&track=example"
	raw_query = request.URL.RawQuery
	if raw_query != expected {
		t.Errorf("Query parameter incorrect, got %#v, expected %#v", raw_query, expected)
	}
}

func TestNonceOverride(t *testing.T) {
	api_url := "https://example.com/endpoint"
	request, _ := http.NewRequest("GET", api_url, nil)
	request.Header.Set("X-OAuth-Nonce", "12345")
	service.Sign(request, user)
	if request.Header.Get("X-OAuth-Nonce") != "" {
		t.Errorf("Nonce override should be cleared after signing")
	}
	header := request.Header.Get("Authorization")
	if !strings.Contains(header, "oauth_nonce=\"12345\"") {
		t.Errorf("Nonce override was not used")
	}
	if strings.Contains(header, "oauth_timestamp=\"\"") {
		t.Errorf("Timestamp not sent when nonce override used")
	}
}

func TestTimestampOverride(t *testing.T) {
	api_url := "https://example.com/endpoint"
	request, _ := http.NewRequest("GET", api_url, nil)
	request.Header.Set("X-OAuth-Timestamp", "54321")
	service.Sign(request, user)
	if request.Header.Get("X-OAuth-Timestamp") != "" {
		t.Errorf("Timestamp override should be cleared after signing")
	}
	header := request.Header.Get("Authorization")
	if !strings.Contains(header, "oauth_timestamp=\"54321\"") {
		t.Errorf("Timestamp override was not used")
	}
	if strings.Contains(header, "oauth_nonce=\"\"") {
		t.Errorf("Nonce not sent when timestamp override used")
	}
}

var ESCAPE_TESTS = map[string]string{
	"aaaa":   "aaaa",
	"Ā":      "%C4%80",
	"Ā㤹":     "%C4%80%E3%A4%B9",
	"bbĀ㤹":   "bb%C4%80%E3%A4%B9",
	"Ā㤹bb":   "%C4%80%E3%A4%B9bb",
	"bbĀ㤹bb": "bb%C4%80%E3%A4%B9bb",
	"㤹":      "%E3%A4%B9",
	"\n":     "%0A",
	"\r":     "%0D",
}

func TestEscaping(t *testing.T) {
	for str, expected := range ESCAPE_TESTS {
		actual := Rfc3986Escape(str)
		if actual != expected {
			t.Errorf("Escaped %v was %v, expected %v", str, actual, expected)
		}
	}
}
