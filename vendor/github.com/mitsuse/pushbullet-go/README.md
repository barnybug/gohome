# pushbullet-go

[![License](https://img.shields.io/badge/license-MIT-yellowgreen.svg?style=flat-square)][license]
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)][godoc]
[![Wercker](http://img.shields.io/wercker/ci/54eb41b6d9b14636631c567f.svg?style=flat-square)][wercker]

[license]: LICENSE.txt
[godoc]: http://godoc.org/github.com/mitsuse/pushbullet-go
[wercker]: https://app.wercker.com/project/bykey/2153719836dc1ecc109b8daf75beb7e1

A library to call [Pushbullet HTTP API](https://docs.pushbullet.com/#http) for Golang. 

## Installation

Just execute the following command:

```bash
$ go get -u github.com/mitsuse/pushbullet-go/...
```

## Example

This is an example to send a simple note via Pushbullet.

```go
package main

import (
	"fmt"
	"os"

	"github.com/mitsuse/pushbullet-go"
	"github.com/mitsuse/pushbullet-go/requests"
)

func main() {
	// Set the access token.
	token := ""

	// Create a client for Pushbullet.
	pb := pushbullet.New(token)

	// Create a push. The following codes create a note, which is one of push types.
	n := requests.NewNote()
	n.Title = "Hello, world!"
	n.Body = "Send via Pushbullet."

	// Send the note via Pushbullet.
	if _, err := pb.PostPushesNote(n); err != nil {
		fmt.Frintf(os.Stderr, "error: %s\n", err)
		return
	}
}
```

## License

The MIT License (MIT)

Copyright (c) 2015 Tomoya Kose.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
