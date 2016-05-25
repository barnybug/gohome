package pushbullet

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitsuse/pushbullet-go/responses"
)

// Post the information of file and get the information of upload destination.
func (pb *Pushbullet) PostUploadRequest(name, mime string) (*responses.Upload, error) {
	res, err := pb.postUploadRequest(name, mime)
	if err != nil {
		return nil, err
	}

	// TODO: Return an error value with human friendly message.
	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}

	decoder := json.NewDecoder(res.Body)

	var upload *responses.Upload
	if err := decoder.Decode(&upload); err != nil {
		return nil, err
	}

	return upload, nil
}

func (pb *Pushbullet) postUploadRequest(name, mime string) (*http.Response, error) {
	values := url.Values{
		"file_name": []string{name},
		"file_type": []string{mime},
	}
	reader := strings.NewReader(values.Encode())

	req, err := http.NewRequest("POST", ENDPOINT_UPLOADREQ, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(pb.token, "")

	return pb.client.Do(req)
}

// Upload a file to S3 specified with the response of PostUploadRequest.
func Upload(client *http.Client, upload *responses.Upload, reader io.Reader) error {
	req, err := createUploadReq(upload, reader)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	// TODO: Return an error value with human friendly message.
	if res.StatusCode < 200 || 300 <= res.StatusCode {
		return errors.New(res.Status)
	}

	return nil
}

func createUploadReq(upload *responses.Upload, reader io.Reader) (*http.Request, error) {
	dest := upload.Data

	buffer := &bytes.Buffer{}

	writer := newMultipartWriter(buffer)

	writer.WriteField("awsaccesskeyid", dest.AwsAccessKeyId)
	writer.WriteField("acl", dest.Acl)
	writer.WriteField("key", dest.Key)
	writer.WriteField("signature", dest.Signature)
	writer.WriteField("policy", dest.Policy)
	writer.WriteField("content-type", dest.ContentType)

	if err := writer.Error(); err != nil {
		return nil, err
	}

	fw, err := writer.CreateFormFile("file", upload.FileName)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(fw, reader); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", upload.UploadUrl, buffer)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())

	return req, nil
}
