package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/barnybug/gohome/pubsub"
)

func fmtFatalf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	os.Exit(1)
}

func httpClient() *http.Client {
	roots := x509.NewCertPool()

	// add system certs
	pemCerts, err := ioutil.ReadFile("/etc/ssl/cert.pem")
	if err == nil {
		roots.AppendCertsFromPEM(pemCerts)
	}

	// add any custom ca certs
	cafile := os.Getenv("GOHOME_CA_CERT")
	if cafile != "" {
		pemCerts, err := ioutil.ReadFile(cafile)
		if err != nil {
			fmtFatalf("Couldn't load CA cert: %s", err)
		}
		roots.AppendCertsFromPEM(pemCerts)
	}

	config := &tls.Config{RootCAs: roots}
	tr := &http.Transport{TLSClientConfig: config}
	return &http.Client{Transport: tr}
}

func request(path string, params url.Values) {
	if os.Getenv("GOHOME_API") == "" {
		fmtFatalf("Set GOHOME_API to the gohome api url.")
	}
	// add http auth
	api := os.Getenv("GOHOME_API")
	uri := fmt.Sprintf("%s/%s", api, path)
	if len(params) > 0 {
		uri += "?" + params.Encode()
	}
	client := httpClient()
	resp, err := client.Get(uri)
	if err != nil {
		if strings.HasSuffix(err.Error(), " EOF") { // yuck
			fmtFatalf("Server disconnected\n")
		} else {
			fmtFatalf("error: %s\n", err)
		}
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)

	n := 0
	for scanner.Scan() {
		ev := pubsub.Parse(scanner.Text())
		if ev == nil {
			continue
		}
		source := ev.Source()
		message := ev.StringField("message")

		if strings.Contains(message, "\n") {
			fmt.Printf("\x1b[32;1m%s\x1b[0m\n%s\n", source, message)
		} else {
			fmt.Printf("\x1b[32;1m%s\x1b[0m %s\n", source, message)
		}
		n += 1
	}
	if n == 0 {
		fmt.Println("No response")
	}
}

func query(first string, rest []string, params url.Values) {
	q := strings.Join(rest, " ")
	u := url.Values{"q": {q}}
	for key, value := range params {
		u[key] = value
	}

	path := fmt.Sprintf("query/%s", first)
	request(path, u)
}
