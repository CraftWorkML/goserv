package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	cfg "goserv/src/configuration"
	"io"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

type (
	ExternalHandler struct {
		mlHost      string
		mlHostAudio string
		mlHostTS    string
		timeout     time.Duration
	}

	PostTrackBody struct {
		Message string `json:"message"`
	}

	PostMlBody struct {
		Message string `json:"message"`
	}

	TSResponseBody struct {
		Datas       []string  `json:"datas"`
		Values      []float32 `json:"values"`
		Futures     []string  `json:"futures"`
		Predictions []float32 `json:"predictions"`
	}
)

const (
	contentTypeImage = "image/png"
	contentTypeAudio = "audio/wav"
)

func NewExternalHandler(config *cfg.Properties) *ExternalHandler {

	return &ExternalHandler{

		mlHostTS:    config.MLServer.HostTS,
		mlHost:      config.MLServer.Host,
		mlHostAudio: config.MLServer.HostAudio,
		timeout:     config.Server.ReadTimeout,
	}
}

func (e *ExternalHandler) SendImageToML(c *gin.Context) {
	requestBody := PostImageBody{Message: c.PostForm("message")}
	// Parse the form data, including the uploaded file
	requestParams := []any{
		map[string]string{"message": requestBody.Message},
		"filedata",
		"test.png"}
	result := e.sendFormHelper(
		c,
		"POST",
		fmt.Sprintf("%s/image", e.mlHost),
		[]string{"image"},
		requestParams)
	// Return the processed image
	if result != nil {
		c.Data(http.StatusOK, contentTypeImage, result.([]byte))
	}
}

func (e *ExternalHandler) SendMessageToML(c *gin.Context) {
	var requestBody PostMlBody
	if err := c.BindJSON(&requestBody); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "no user in query"})
		return
	}
	parsedJSON, err := json.Marshal(&requestBody)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "can not marshal JSON:", "error": err.Error()})
		return
	}
	requestParams := []any{parsedJSON}
	result := e.sendJSONHelper(
		c,
		"POST",
		fmt.Sprintf("%s/message", e.mlHost),
		requestParams)

	// Return the processed image
	if result != nil {
		c.Data(http.StatusOK, contentTypeImage, result.([]byte))
	}
}

func (e *ExternalHandler) SendTrackToML(c *gin.Context) {
	var requestBody PostTrackBody
	if err := c.BindJSON(&requestBody); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no user in query"})
		return
	}
	parsedJSON, err := json.Marshal(&requestBody)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "can not marshal JSON", "error": err.Error()})
		return
	}
	requestParams := []any{parsedJSON}
	result := e.sendJSONHelper(
		c,
		"POST",
		fmt.Sprintf("%s/track", e.mlHostAudio),
		requestParams)

	// Return the processed image
	if result != nil {
		c.Data(http.StatusOK, contentTypeAudio, result.([]byte))
	}
}

func (e *ExternalHandler) SendMelodyToML(c *gin.Context) {
	requestBody := PostTrackBody{Message: c.PostForm("message")}
	// Parse the form data, including the uploaded file
	requestParams := []any{
		map[string]string{"message": requestBody.Message},
		"filedata",
		"test.wav"}
	result := e.sendFormHelper(
		c,
		"POST",
		fmt.Sprintf("%s/melody", e.mlHostAudio),
		[]string{"audio"},
		requestParams)
	if result != nil {
		c.Data(http.StatusOK, contentTypeAudio, result.([]byte))

	}
}

func (e *ExternalHandler) SendTSToML(c *gin.Context) {
	predictor := c.PostForm("predictor")
	target := c.PostForm("target")
	requestParams := []any{
		map[string]string{"predictor": predictor, "target": target},
		"filedata",
		"test.csv"}
	result := e.sendFormHelper(
		c,
		"POST",
		fmt.Sprintf("%s/ts", e.mlHostTS),
		[]string{"ts"},
		requestParams)
	if result != nil {
		result, err := postProcTS(result.([]byte))
		if err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "can not marshal JSON", "error": err.Error()})
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "arrays": result.([]byte)})

	}
}

func (e *ExternalHandler) sendFormHelper(
	c *gin.Context,
	restCmd string,
	hostname string,
	filenames []string,
	reqParam []any) any {
	var buffer bytes.Buffer
	// Parse the form data, including the uploaded file
	for _, filename := range filenames {
		file, _, err := c.Request.FormFile(filename)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		defer file.Close()
		// Read the file into a buffer
		_, err = io.Copy(&buffer, file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		}
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), e.timeout)
	defer func() {
		recovered := recover()
		if recovered != nil {
			debug.PrintStack()
			err := fmt.Errorf("recovered from: %s", recovered)
			fmt.Println(err)
		}
	}()
	defer cancel()
	result := make(chan any)
	errors := make(chan error)
	requestPipe := RequestPipeline{
		parametersParser: prepareMultipartFile,
		transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    e.timeout,
			DisableCompression: true,
		},
		postProcess: nil,
	}
	requestParams := append(reqParam, &buffer)
	go requestPipe.Execute(
		restCmd,
		hostname,
		e.timeout,
		result,
		errors,
		requestParams)

	select {
	case response := <-result:
		return response.([]byte)
	case err := <-errors:
		c.IndentedJSON(
			http.StatusInternalServerError,
			gin.H{"message": "response error", "error": err.Error()})
	case <-ctx.Done():
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": "timeout from server", "error": fmt.Sprintf("%s", hostname)})
	}
	return nil
}

func (e *ExternalHandler) sendJSONHelper(
	c *gin.Context,
	restCmd string,
	hostname string,
	reqParam []any) any {
	ctx, cancel := context.WithTimeout(c.Request.Context(), e.timeout)
	defer func() {
		recovered := recover()
		if recovered != nil {
			debug.PrintStack()
			err := fmt.Errorf("recovered from: %s", recovered)
			fmt.Println(err)
		}
	}()
	defer cancel()
	result := make(chan any)
	errors := make(chan error)
	requestPipe := RequestPipeline{
		parametersParser: prepareJSONBody,
		transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    e.timeout,
			DisableCompression: true,
		},
		postProcess: nil,
	}

	go requestPipe.Execute(
		restCmd,
		hostname,
		e.timeout,
		result,
		errors,
		reqParam)
	select {
	case response := <-result:
		return response.([]byte)
	case err := <-errors:
		c.IndentedJSON(
			http.StatusInternalServerError,
			gin.H{"message": "response error", "error": err.Error()})
	case <-ctx.Done():
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": "timeout", "error": fmt.Sprintf("timeout from server: %s", hostname)})
	}
	return nil
}

func postProcTS(responseBody []byte) (any, error) {
	ts := TSResponseBody{}
	err := json.Unmarshal(responseBody, &ts)
	if err != nil {
		return nil, fmt.Errorf("can not unmarshall time-series: %e", err)
	}
	return ts, nil
}
