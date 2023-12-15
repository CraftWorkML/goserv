package server

import (
	"bytes"
	"fmt"
	app "goserv/src/app"
	cfg "goserv/src/configuration"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	AppHandler struct {
		s3 *app.MinioS3Client
	}

	PostImageBody struct {
		Message string `json:"message"`
	}

	ResponseListImage struct {
		Images []string `json:"images"`
	}

	DeleteImageBody struct {
		User string `json:"user"`
		Name string `json:"name"`
	}
)

const (
	userQueryParam = "user"
)

var (
	imageAvaiableFormats = []string{"png", "jpg", "tiff", "bmp"}
	audioAvaiableFormats = []string{"mp3", "wav", "fb2", "midi"}
)

func NewS3Handler(config *cfg.Properties, s3Client *app.MinioS3Client) *AppHandler {

	return &AppHandler{
		s3: s3Client,
	}

}

func (a *AppHandler) GetImageList(c *gin.Context) {
	user, ok := c.GetQuery(userQueryParam)
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "error", "error": "no user in query"})
		return
	}
	result := []string{}
	images, err := a.s3.ListObjects(user, imageAvaiableFormats)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": "error", "error": fmt.Errorf("can not fetch images from s3: %e", err).Error()})

		return
	}
	for _, image := range images {
		result = append(result, image.String())
	}

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": "error", "error": fmt.Errorf("can not marshall result: %e", err).Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "payload": result})
}

func (a *AppHandler) GetAudioList(c *gin.Context) {
	user, ok := c.GetQuery(userQueryParam)
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "no user in query"})
		return
	}
	result := []string{}
	tracks, err := a.s3.ListObjects(user, audioAvaiableFormats)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": "error", "error": fmt.Errorf("can not fetch images from s3: %e", err).Error()})

		return
	}
	for _, track := range tracks {
		result = append(result, track.String())
	}

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": "error", "error": fmt.Errorf("can not marshall result: %e", err).Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "payload": result})
}

func (a *AppHandler) PostImage(c *gin.Context) {

	// Parse the form data, including the uploaded file
	file, _, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "error", "error": fmt.Errorf("can not find image in request: %e", err).Error()})
		return
	}
	defer file.Close()

	// Read the file into a buffer
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "error", "error": fmt.Errorf("Failed to read file:%e", err).Error()})
		return
	}
	if err := a.s3.UploadFile(fmt.Sprintf("%s/%s", c.PostForm("user"), c.PostForm("name")),
		&buffer,
		buffer.Len()); err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": "error", "error": fmt.Errorf("can not upload image to s3: %e", err).Error()})

	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (a *AppHandler) DeleteImage(c *gin.Context) {
	var requestBody DeleteImageBody
	if err := c.BindJSON(&requestBody); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "error", "error": fmt.Errorf("cannot delete: %e", err).Error()})
		return
	}

	if err := a.s3.DeleteFile(fmt.Sprintf("%s/%s", requestBody.User, requestBody.Name)); err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": "error", "error": fmt.Errorf("can not delete image from s3: %e", err).Error()})

	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
