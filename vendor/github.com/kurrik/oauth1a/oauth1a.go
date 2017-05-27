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

/*
	Package oauth1a implements the OAuth 1.0a specification.
*/
package oauth1a

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Container for client-specific configuration related to the OAuth process.
// This struct is intended to be serialized and stored for future use.
type ClientConfig struct {
	ConsumerSecret string
	ConsumerKey    string
	CallbackURL    string
}

// Represents an API which offers OAuth access.
type Service struct {
	RequestURL   string
	AuthorizeURL string
	AccessURL    string
	*ClientConfig
	Signer
}

// Signs an HTTP request with the needed OAuth parameters.
func (s *Service) Sign(request *http.Request, userConfig *UserConfig) error {
	return s.Signer.Sign(request, s.ClientConfig, userConfig)
}

// Interface for any OAuth signing implementations.
type Signer interface {
	Sign(request *http.Request, config *ClientConfig, user *UserConfig) error
}

// A Signer which implements the HMAC-SHA1 signing algorithm.
type HmacSha1Signer struct{}

// Generate a unique nonce value.
func (HmacSha1Signer) generateNonce() string {
	nonce := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, nonce)
	if err != nil {
		panic("unable to read from crypto/rand.Reader; process is insecure: " + err.Error())
	}
	return hex.EncodeToString(nonce)
}

// Generate a timestamp.
func (HmacSha1Signer) generateTimestamp() int64 {
	return time.Now().UTC().Unix()
}

// Returns a map of all of the oauth_* (including signature) parameters for the
// given request, and the signature base string used to generate the signature.
func (s *HmacSha1Signer) GetOAuthParams(request *http.Request, clientConfig *ClientConfig, userConfig *UserConfig, nonce string, timestamp string) (map[string]string, string) {
	oauthParams := map[string]string{
		"oauth_consumer_key":     clientConfig.ConsumerKey,
		"oauth_nonce":            nonce,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        timestamp,
		"oauth_version":          "1.0",
	}
	tokenKey, tokenSecret := userConfig.GetToken()
	if tokenKey != "" {
		oauthParams["oauth_token"] = tokenKey
	}

	signingParams := request.URL.Query()
	for key, value := range oauthParams {
		signingParams.Set(key, value)
	}

	if request.Body != nil && request.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		request.ParseForm()
		for key, values := range request.Form {
			for _, v := range values {
				signingParams.Add(key, v)
			}
		}
		// Calling ParseForm clears out the reader.  It may be
		// necessary to do this in a less destructive way, but for
		// right now, this code reinitializes the body of the request.
		body := strings.NewReader(request.Form.Encode())
		request.Body = ioutil.NopCloser(body)
		request.ContentLength = int64(body.Len())
	}

	signingUrl := fmt.Sprintf("%v://%v%v", request.URL.Scheme, request.URL.Host, request.URL.Path)
	signatureParts := []string{
		request.Method,
		url.QueryEscape(signingUrl),
		url.QueryEscape(sortedQueryString(signingParams))}
	signatureBase := strings.Join(signatureParts, "&")
	oauthParams["oauth_signature"] = s.GetSignature(clientConfig.ConsumerSecret, tokenSecret, signatureBase)
	return oauthParams, signatureBase
}

// Calculates the HMAC-SHA1 signature of a base string, given a consumer and
// token secret.
func (s *HmacSha1Signer) GetSignature(consumerSecret string, tokenSecret string, signatureBase string) string {
	signingKey := consumerSecret + "&" + tokenSecret
	signer := hmac.New(sha1.New, []byte(signingKey))
	signer.Write([]byte(signatureBase))
	oauthSignature := base64.StdEncoding.EncodeToString(signer.Sum(nil))
	return oauthSignature
}

// Given an unsigned request, add the appropriate OAuth Authorization header
// using the HMAC-SHA1 algorithm.
func (s *HmacSha1Signer) Sign(request *http.Request, clientConfig *ClientConfig, userConfig *UserConfig) error {
	var (
		nonce     string
		timestamp string
	)
	if nonce = request.Header.Get("X-OAuth-Nonce"); nonce != "" {
		request.Header.Del("X-OAuth-Nonce")
	} else {
		nonce = s.generateNonce()
	}
	if timestamp = request.Header.Get("X-OAuth-Timestamp"); timestamp != "" {
		request.Header.Del("X-OAuth-Timestamp")
	} else {
		timestamp = fmt.Sprintf("%v", s.generateTimestamp())
	}
	oauthParams, _ := s.GetOAuthParams(request, clientConfig, userConfig, nonce, timestamp)

	headerParts := make([]string, 0, len(oauthParams))
	for key, value := range oauthParams {
		val := Rfc3986Escape(key) + "=\"" + Rfc3986Escape(value) + "\""
		headerParts = append(headerParts, val)
	}
	sort.Strings(headerParts)
	oauthHeader := "OAuth " + strings.Join(headerParts, ", ")
	request.Header["Authorization"] = []string{oauthHeader}

	request.URL.RawQuery = sortedQueryString(request.URL.Query())
	return nil
}

// This bit of fussing is because '/' is not encoded correctly
// by the URL package, so we encode manually.
func sortedQueryString(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	pairs := make(sortedPairs, 0)
	for k, vs := range values {
		escK := Rfc3986Escape(k)
		for _, v := range vs {
			pairs = append(pairs, pair{escK, Rfc3986Escape(v)})
		}
	}
	sort.Sort(pairs)

	buf := new(bytes.Buffer)
	buf.WriteString(pairs[0].k)
	buf.WriteByte('=')
	buf.WriteString(pairs[0].v)

	for _, p := range pairs[1:] {
		buf.WriteByte('&')
		buf.WriteString(p.k)
		buf.WriteByte('=')
		buf.WriteString(p.v)
	}
	return buf.String()
}

type pair struct{ k, v string }
type sortedPairs []pair

func (sp sortedPairs) Len() int {
	return len(sp)
}
func (p sortedPairs) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p sortedPairs) Less(i, j int) bool {
	if p[i].k == p[j].k {
		return p[i].v < p[j].v
	}
	return p[i].k < p[j].k
}

// Escapes a string more in line with Rfc3986 than http.URLEscape.
// URLEscape was converting spaces to "+" instead of "%20", which was messing up
// the signing of requests.
func Rfc3986Escape(input string) string {
	firstEsc := -1
	b := []byte(input)
	for i, c := range b {
		if !isSafeChar(c) {
			firstEsc = i
			break
		}
	}

	// If nothing needed to be escaped, then the input is clean and
	// we're done.
	if firstEsc == -1 {
		return input
	}

	// If something did need to be escaped, write the prefix that was
	// fine to the buffer and iterate through the rest of the bytes.
	output := new(bytes.Buffer)
	output.Write(b[:firstEsc])

	for _, c := range b[firstEsc:] {
		if isSafeChar(c) {
			output.WriteByte(c)
		} else {
			fmt.Fprintf(output, "%%%02X", c)
		}
	}
	return output.String()
}

func isSafeChar(c byte) bool {
	return ('0' <= c && c <= '9') ||
		('a' <= c && c <= 'z') ||
		('A' <= c && c <= 'Z') ||
		c == '-' || c == '.' || c == '_' || c == '~'
}
