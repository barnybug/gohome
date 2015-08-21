// Copyright 2012 Arne Roomann-Kurrik
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

package json

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestParseString(t *testing.T) {
	var (
		gold    = "Hello world"
		encoded = []byte(fmt.Sprintf("\"%v\"", gold))
		parsed  string
	)
	if err := Unmarshal(encoded, &parsed); err != nil {
		t.Fatalf("%v", err)
	}

	if gold != parsed {
		t.Fatalf("%v != %v", gold, parsed)
	}
}

func TestParseNumber(t *testing.T) {
	var (
		gold    int64 = 1234567
		encoded       = []byte(fmt.Sprintf("%v", gold))
		parsed  int64
	)
	if err := Unmarshal(encoded, &parsed); err != nil {
		t.Fatalf("%v", err)
	}
	if gold != parsed {
		t.Fatalf("%v != %v", gold, parsed)
	}
}

func TestParseNegativeNumber(t *testing.T) {
	var (
		gold    int64 = -1234567
		encoded       = []byte(fmt.Sprintf("%v", gold))
		parsed  int64
	)
	if err := Unmarshal(encoded, &parsed); err != nil {
		t.Fatalf("%v", err)
	}
	if gold != parsed {
		t.Fatalf("%v != %v", gold, parsed)
	}
}

func TestParseFloat(t *testing.T) {
	var (
		gold    float64 = 1234567.89
		encoded         = []byte("1234567.89")
		parsed  float64
	)
	if err := Unmarshal(encoded, &parsed); err != nil {
		t.Fatalf("%v", err)
	}
	if gold != parsed {
		t.Fatalf("%v != %v", gold, parsed)
	}
}

func TestParseMap(t *testing.T) {
	var (
		gold = map[string]interface{}{
			"foo": "Bar",
			"baz": 1234,
		}
		encoded = []byte("{\"foo\":\"Bar\",\"baz\":1234}")
		parsed  map[string]interface{}
	)
	if err := Unmarshal(encoded, &parsed); err != nil {
		t.Fatalf("%v", err)
	}
	if len(parsed) != len(gold) {
		t.Fatalf("Parsed len %v != gold len %v", len(parsed), len(gold))
	}
	for i, v := range parsed {
		if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", gold[i]) {
			t.Errorf("%v: %v != %v", i, v, gold[i])
		}
	}
}

func TestParseArray(t *testing.T) {
	var (
		gold    = []interface{}{1234, "Foo", 5678}
		encoded = []byte("[1234,\"Foo\",5678]")
		parsed  []interface{}
	)
	if err := Unmarshal(encoded, &parsed); err != nil {
		t.Fatalf("%v", err)
	}
	if len(parsed) != len(gold) {
		t.Fatalf("Parsed len %v != gold len %v", len(parsed), len(gold))
	}
	for i, v := range parsed {
		if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", gold[i]) {
			t.Errorf("%v: %v != %v", i, v, gold[i])
		}
	}
}

type Bucket map[string]interface{}
type BucketList []Bucket

func TestParseStruct(t *testing.T) {
	var (
		encoded = []byte("[{\"foo\":1},{\"foo\":2}]")
		gold    = BucketList{Bucket{"foo": 1}, Bucket{"foo": 2}}
		parsed  BucketList
	)
	if err := Unmarshal(encoded, &parsed); err != nil {
		t.Fatalf("%v", err)
	}
	if len(parsed) != len(gold) {
		t.Fatalf("Parsed len %v != gold len %v", len(parsed), len(gold))
	}
	for i, v := range parsed {
		if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", gold[i]) {
			t.Errorf("%v: %v != %v", i, v, gold[i])
		}
	}
}

func TestParseTwitterUser(t *testing.T) {
	var (
		parsed map[string]interface{}
		status map[string]interface{}
		raw    []byte
		err    error
		path   string
	)
	path = "data/twitter_user.json"
	if raw, err = ioutil.ReadFile(path); err != nil {
		t.Fatalf("Could not read data file: %v", path)
	}
	if err = Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Could not parse Twitter user: %v", err)
	}
	if parsed["id"] != int64(370773112) {
		t.Fatalf("Could not parse 64-bit Twitter user ID.")
	}
	if parsed["name"] != "fakekurrik" {
		t.Fatalf("Could not parse Twitter user screen name.")
	}
	status = parsed["status"].(map[string]interface{})
	if status["id"] != int64(291983420479905792) {
		t.Fatalf("Could not parse nested 64-bit Tweet ID.")
	}
}

func TestParseTwitterTimeline(t *testing.T) {
	var (
		parsed []interface{}
		status map[string]interface{}
		raw    []byte
		err    error
		path   string
	)
	path = "data/twitter_timeline.json"
	if raw, err = ioutil.ReadFile(path); err != nil {
		t.Fatalf("Could not read data file: %v", path)
	}
	if err = Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Could not parse Twitter timeline: %v", err)
	}
	if len(parsed) != 100 {
		t.Fatalf("Expected 100 nested Tweets, got: %v", len(parsed))
	}
	status = parsed[0].(map[string]interface{})
	if status["text"].(string)[0:15] != "We are updating" {
		t.Fatalf("Could not parse nested Tweet text.")
	}
}

type Tweet map[string]interface{}
type Timeline []Tweet

func TestParseTwitterTweetToType(t *testing.T) {
	var (
		parsed = &Tweet{}
		raw    []byte
		err    error
		path   string
		place  map[string]interface{}
		bbox   map[string]interface{}
		coord  []interface{}
		value  float64
		gold   float64
		attr   map[string]interface{}
	)
	path = "data/twitter_tweet.json"
	if raw, err = ioutil.ReadFile(path); err != nil {
		t.Fatalf("Could not read data file: %v", path)
	}
	if err = Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Could not parse Twitter Tweet: %v", err)
	}
	place = (*parsed)["place"].(map[string]interface{})
	bbox = place["bounding_box"].(map[string]interface{})
	coord = bbox["coordinates"].([]interface{})[0].([]interface{})

	value = coord[0].([]interface{})[0].(float64)
	gold = -122.513682
	if value != gold {
		t.Fatalf("Bad coord: %v, wanted : %v", value, gold)
	}

	attr = place["attributes"].(map[string]interface{})
	if len(attr) != 0 {
		t.Fatalf("Wrong length for empty attributes object")
	}
}

func TestParseTwitterTimelineToType(t *testing.T) {
	var (
		parsed = &Timeline{}
		raw    []byte
		err    error
		path   string
		gold   string
		value  string
	)
	path = "data/twitter_timeline.json"
	if raw, err = ioutil.ReadFile(path); err != nil {
		t.Fatalf("Could not read data file: %v", path)
	}
	if err = Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Could not parse Twitter timeline: %v", err)
	}
	if len(*parsed) != 100 {
		t.Fatalf("Expected 100 nested Tweets, got: %v", len(*parsed))
	}
	if (*parsed)[0]["text"].(string)[0:15] != "We are updating" {
		t.Fatalf("Could not parse nested structured Tweet text.")
	}
	gold = "210090093417992192"
	value = (*parsed)[56]["id_str"].(string)
	if value != gold {
		t.Fatalf("Tweet 57 has ID %v, expected %v.", value, gold)
	}
}

func TestParseEmptyTwitterTimelineToType(t *testing.T) {
	var (
		parsed = Timeline{}
		raw    []byte
		err    error
	)
	raw = []byte("[]")
	if err = Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Could not parse Twitter timeline: %v", err)
	}
	if len(parsed) != 0 {
		t.Fatalf("Expected 0 Tweets, got: %v", len(parsed))
	}
}

func TestDecode(t *testing.T) {
	var (
		parsed map[string]interface{}
		path   string
		paths  []string
		raw    []byte
		err    error
	)
	paths = []string{
		"data/twitter_tweet2.json",
		"data/twitter_tweet3.json",
		"data/twitter_tweet4.json",
		"data/twitter_tweet5.json",
		"data/twitter_tweet6.json",
		"data/twitter_tweet7.json",
	}
	for _, path = range paths {
		if raw, err = ioutil.ReadFile(path); err != nil {
			t.Fatalf("Could not read data file: %v", path)
		}
		if err = Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("Could not parse %v: %v", path, err)
		}
	}
}
