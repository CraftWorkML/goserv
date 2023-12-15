package server

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type (
	// RequestPipeline defines a pipeline for executing HTTP requests with specified parameters
	RequestPipeline struct {
		parametersParser func(restCmd string, endPoint string, params ...any) (*http.Request, error)
		transport        *http.Transport
		postProcess      func(responseBody []byte) (any, error)
	}
)
// Execute performs the HTTP request according to the configured pipeline
func (r RequestPipeline) Execute(
	restCmd string,
	endPoint string,
	timeout time.Duration,
	result chan any,
	errors chan error,
	params []any) {
	request, err := r.parametersParser(restCmd, endPoint, params...)
	if err != nil {
		errors <- fmt.Errorf("error during prepare request: %e", err)
	} else {
		client := &http.Client{
			Transport: r.transport,
			Timeout:   timeout}
		resp, err := client.Do(request)
		//TODO: Handle errors from ml server
		if err != nil {
			errors <- fmt.Errorf("error during request to the host: %e", err)
		}
		defer resp.Body.Close()
		// Read the processed image into a byte slice
		processedBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			errors <- fmt.Errorf("error during body response: %e", err)
		}
		if r.postProcess != nil {
			res, err := r.postProcess(processedBytes)
			if err != nil {
				errors <- fmt.Errorf("error during  response parse: %e", err)
			} else {
				result <- res
			}
		} else {
			result <- processedBytes
		}
	}
}

func prepareMultipartFile(restCmd string, endPoint string, params ...any) (*http.Request, error) {
	bodyReader := new(bytes.Buffer)
	writer := multipart.NewWriter(bodyReader)
	fields := params[0].(map[string]string)
	for fieldName, fieldValue := range fields {
		fw, err := writer.CreateFormField(fieldName)
		if err != nil {
			return nil, fmt.Errorf("can not marshall body: %e", err)
		}
		_, err = io.Copy(fw, strings.NewReader(fieldValue))
		if err != nil {
			return nil, fmt.Errorf("can not marshall body: %e", err)
		}
	}
	fileLabel := params[1].(string)
	fileName := params[2].(string)
	buf := params[3].(*bytes.Buffer)
	part, err := writer.CreateFormFile(fileLabel, fileName)
	if err != nil {
		return nil, fmt.Errorf("can not marshall body: %e", err)
	}
	part.Write(buf.Bytes())
	writer.Close()
	req, err := http.NewRequest(restCmd, endPoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("can not build a request: %e", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func prepareJSONBody(restCmd string, endPoint string, params ...any) (*http.Request, error) {
	json := params[0].([]byte)
	bodyReader := bytes.NewReader(json)
	req, err := http.NewRequest(restCmd, endPoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("can not build a request: %e", err)
	}
	return req, nil
}
