package server

import (
	"fmt"
	"io"
	"net/http"
)

type (
	RequestPipeline struct {
		parametersParser func(params any) (io.Reader, error)
		transport        *http.Transport
		requestPrepare   func(body io.Reader, params any) (*http.Request, error)
		postProcess      func(responseBody []byte, params any) (any, error)
	}
)

func (r RequestPipeline) Execute(result chan any,
	errors chan error,
	paramChan <-chan any,
	reqChan <-chan any,
	postChan <-chan any) {
	params := <-paramChan
	reader, err := r.parametersParser(params)
	if err != nil {
		errors <- fmt.Errorf("error during prepare: %e", err)
	} else {
		reqParam := <-reqChan
		request, err := r.requestPrepare(reader, reqParam)
		if err != nil {
			errors <- fmt.Errorf("error during request prepare: %e", err)
		} else {
			client := &http.Client{Transport: r.transport}
			resp, err := client.Do(request)
			//TODO: Handle errors from ml server
			if err != nil {
				errors <- fmt.Errorf("error during request sending: %e", err)
			}
			defer resp.Body.Close()
			// Read the processed image into a byte slice
			processedBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				errors <- fmt.Errorf("error during body response: %e", err)
			}
			postParams := <-postChan
			res, err := r.postProcess(processedBytes, postParams)
			if err != nil {
				errors <- fmt.Errorf("error during body response: %e", err)
			} else {
				result <- res
			}
		}
	}
}
