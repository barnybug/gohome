package pushbullet

import (
	"io"
	"mime/multipart"
	"net/textproto"
)

type multipartWriter struct {
	writer *multipart.Writer
	err    error
}

func newMultipartWriter(writer io.Writer) *multipartWriter {
	m := &multipartWriter{
		writer: multipart.NewWriter(writer),
	}

	return m
}

func (m *multipartWriter) Error() error {
	return m.err
}

func (m *multipartWriter) Boundary() string {
	return m.writer.Boundary()
}

func (m *multipartWriter) Close() error {
	return m.writer.Close()
}

func (m *multipartWriter) CreateFormField(fieldname string) (io.Writer, error) {
	if m.err != nil {
		return nil, m.err
	}

	fw, err := m.writer.CreateFormField(fieldname)
	m.err = err

	return fw, err
}

func (m *multipartWriter) CreateFormFile(fieldname, filename string) (io.Writer, error) {
	if m.err != nil {
		return nil, m.err
	}

	fw, err := m.writer.CreateFormFile(fieldname, filename)
	m.err = err

	return fw, err
}

func (m *multipartWriter) CreatePart(header textproto.MIMEHeader) (io.Writer, error) {
	if m.err != nil {
		return nil, m.err
	}

	pw, err := m.writer.CreatePart(header)
	m.err = err

	return pw, err
}

func (m *multipartWriter) FormDataContentType() string {
	return m.writer.FormDataContentType()
}

func (m *multipartWriter) SetBoundary(boundary string) error {
	if m.err != nil {
		return m.err
	}

	m.err = m.SetBoundary(boundary)

	return m.err
}

func (m *multipartWriter) WriteField(fieldname, value string) error {
	if m.err != nil {
		return m.err
	}

	m.err = m.writer.WriteField(fieldname, value)

	return m.err
}
