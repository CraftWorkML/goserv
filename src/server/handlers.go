package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	app "goserv/src/app"
	cfg "goserv/src/configuration"
	db "goserv/src/repository"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type (
	AppHandler struct {
		oidcProvider           *oidc.Provider
		dataStore              db.AuthDB
		s3                     *app.MinioS3Client
		mlHost                 string
		mlHostAudio            string
		mlHostTS               string
		AuthConfig             *oauth2.Config
		ClientID               string
		AccessTokenCookieName  string
		RefreshTokenCookieName string
		IDTokenCookieName      string
		oauthStateString       string
	}

	PostImageBody struct {
		Message string `json:"message"`
	}

	PostTrackBody struct {
		Message string `json:"message"`
	}

	ResponseListImage struct {
		Images []string `json:"images"`
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

	DeleteImageBody struct {
		User string `json:"user"`
		Name string `json:"name"`
	}
)

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func NewHandler(config *cfg.Properties, s3Client *app.MinioS3Client) *AppHandler {
	log.Printf("config get %v", config)
	provider, err := oidc.NewProvider(oauth2.NoContext, config.Auth.Host)

	if err != nil {
		fmt.Println("Error creating OIDC provider: " + err.Error())
	} else {
		log.Printf("endpoint is %v", provider.Endpoint())
		// initialize OAuth
		authConfig := &oauth2.Config{
			ClientID:     config.Auth.ID,
			ClientSecret: config.Auth.Secret,
			RedirectURL:  config.Auth.Redirect,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "api"},
		}
		dataConnect, err := db.NewAuthDataBase(config)
		if err != nil {
			log.Fatalf("database not respond %e", err)
			return nil
		}
		if !dataConnect.Connect() {
			log.Fatalf("can not connect to database %e", err)
			return nil

		}

		return &AppHandler{
			s3:                     s3Client,
			mlHostTS:               config.MLServer.HostTS,
			mlHost:                 config.MLServer.Host,
			mlHostAudio:            config.MLServer.HostAudio,
			dataStore:              dataConnect,
			oidcProvider:           provider,
			AuthConfig:             authConfig,
			ClientID:               config.Auth.ID,
			AccessTokenCookieName:  config.Auth.AccessTokenCookieName,
			RefreshTokenCookieName: config.Auth.RefreshTokenCookieName,
			IDTokenCookieName:      config.Auth.IDTokenCookieName,
		}
	}
	return &AppHandler{}
}

func (a *AppHandler) GetHealth(c *gin.Context) {
	// Now get the ID token so we can show the user's email address
	cookie, err := c.Cookie("callback")
	log.Printf("current cookie is %v", cookie)
	if err == nil {
		log.Println(cookie)
	} else {
		log.Printf("No cookie %e found", err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (a *AppHandler) Root(c *gin.Context) {
	if !a.authorize(c) {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "No Authorize to get resourse"})
		return
	}

}

func (a *AppHandler) Login(c *gin.Context) {
	a.oauthStateString, _ = randString(16)
	// Now get the ID token so we can show the user's email address
	callback, _ := c.Cookie("callback")
	log.Printf("1. callback %v", callback)
	log.Print("2. ...")
	origin := c.Request.Header.Get("Origin")
	log.Printf("1. origin %v", origin)
	c.JSON(http.StatusOK, gin.H{"ref": a.AuthConfig.AuthCodeURL(a.oauthStateString)})
	//c.Redirect(http.StatusFound, a.AuthConfig.AuthCodeURL(oauthStateString))
}

func (a *AppHandler) Singin(c *gin.Context) {
	a.oauthStateString, _ = randString(16)
	c.Redirect(http.StatusFound, a.AuthConfig.AuthCodeURL(a.oauthStateString))
}

func (a *AppHandler) Logout(c *gin.Context) {
	c.SetCookie(a.AccessTokenCookieName, "", int(-1), "/", "localhost", false, false)
	c.SetCookie(a.RefreshTokenCookieName, "", int(-1), "/", "localhost", false, false)
	c.SetCookie(a.IDTokenCookieName, "", int(-1), "/", "localhost", false, false)

}

func (a *AppHandler) Callback(c *gin.Context) {
	log.Print("In callback")
	callback, _ := c.Cookie("callback")
	log.Printf("callback %v", callback)
	//origin := c.Request.Header.Get("Origin")
	log.Printf("1. origin %v", c.Request)
	state := c.Query("state")
	//if !ok {
	//	c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no state in query"})
	//	return
	//}
	code := c.Query("code")
	//if !ok {
	//	c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no code in query"})
	//	return
	//}
	if state != a.oauthStateString {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no current state found"})
		return
	}

	// Exchange the authorization code for access, refresh, and id tokens
	token, err := a.AuthConfig.Exchange(oauth2.NoContext, code)

	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Error getting access token: " + err.Error()})
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)

	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "No ID token found in request to /callback"})
		return
	}
	log.Printf("get token t: %v", token)
	log.Printf("get token r: %v", token.Expiry)

	// Write access, refresh, and id tokens to http-only cookies
	c.SetCookie(a.AccessTokenCookieName, token.AccessToken, int(3600), "/", "demoapp.aramcoinnovations.com", false, false)
	c.SetCookie(a.RefreshTokenCookieName, token.RefreshToken, int(3600), "/", "demoapp.aramcoinnovations.com", false, false)
	c.SetCookie(a.IDTokenCookieName, rawIDToken, int(3600), "/", "demoapp.aramcoinnovations.com", false, false)
	a.dataStore.UploadUser(token.AccessToken, token.RefreshToken)
	cookie, err := c.Cookie("callback")
	log.Printf("current cookie is %v", cookie)
	if err == nil {
		log.Println(cookie)
	} else {
		log.Printf("No cookie %e found", err)
	}
	c.Redirect(http.StatusFound, cookie)
}

func (a *AppHandler) Account(c *gin.Context) {
	if !a.authorize(c) {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "No Authorize to get resourse"})
		return
	}
	// Now get the ID token so we can show the user's email address
	cookie, err := c.Cookie(a.IDTokenCookieName)

	if err != nil || cookie == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "No ID token found"})
		return
	}

	var verifier = a.oidcProvider.Verifier(&oidc.Config{ClientID: a.ClientID})

	idToken, err := verifier.Verify(oauth2.NoContext, cookie)

	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Error verifying ID token: " + err.Error()})
		return
	}
	//TODO: check expiration and redirect to /login
	fmt.Printf("token is %v", idToken.Expiry)
	var claims struct {
		Name     string `json:"nickname"`
		Picture  string `json:"picture"`
		Verified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Can not parse claims verifying ID token: " + err.Error()})
		// handle error
	}
	user := app.User{
		ID:      claims.Name,
		Name:    claims.Name,
		Picture: claims.Picture,
	}

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("can not marshall result: %e", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "payload": user})

	fmt.Printf("Claims is %v", user)
}

func (a *AppHandler) GetImageList(c *gin.Context) {
	user, ok := c.GetQuery("user")
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no user in query"})
		return
	}
	result := []string{}
	images, err := a.s3.ListObjects(user, []string{"png", "jpg", "tiff", "bmp"})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("can not fetch images from s3: %e", err)})

		return
	}
	for _, image := range images {
		result = append(result, image.String())
	}

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("can not marshall result: %e", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "payload": result})
}

func (a *AppHandler) GetAudioList(c *gin.Context) {
	user, ok := c.GetQuery("user")
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no user in query"})
		return
	}
	result := []string{}
	tracks, err := a.s3.ListObjects(user, []string{"mp3", "wav", "fb2", "midi"})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("can not fetch images from s3: %e", err)})

		return
	}
	for _, track := range tracks {
		result = append(result, track.String())
	}

	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("can not marshall result: %e", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "payload": result})
}

func (a *AppHandler) PostImage(c *gin.Context) {

	// Parse the form data, including the uploaded file
	log.Printf("req is %v", c.Request)
	file, _, err := c.Request.FormFile("image")
	if err != nil {
		log.Printf("error is %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	// Read the file into a buffer
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	log.Printf("filename is: %s", fmt.Sprintf("%s/%s", c.PostForm("user"), c.PostForm("name")))
	if err := a.s3.UploadFile(fmt.Sprintf("%s/%s", c.PostForm("user"), c.PostForm("name")), &buffer, buffer.Len()); err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("can not upload image to s3: %e", err)})

	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (a *AppHandler) SendImageToML(c *gin.Context) {
	log.Print("MAin Herer")
	requestBody := PostImageBody{Message: c.PostForm("message")}
	var result []byte
	//if err := c.BindJSON(&requestBody); err != nil {
	//	c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no user in query"})
	//	return
	//}
	log.Printf("MAin Herer, %v", c.PostForm("message"))

	// Parse the form data, including the uploaded file
	file, _, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()
	//log.Printf("recieve file to send %v and %v", file, loc)
	// Read the file into a buffer
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second*60*60)
	defer func() {
		recovered := recover()
		if recovered != nil {
			debug.PrintStack()
			err := fmt.Errorf("recovered from: %s", recovered)
			fmt.Println(err)
		}
	}()
	defer cancel()
	process := make(chan []byte)
	go func(buffer *bytes.Buffer, message string, process chan []byte) {
		// Prepare a new HTTP POST request to the processing server
		log.Print("Herer")

		bodyReader := new(bytes.Buffer)
		writer := multipart.NewWriter(bodyReader)
		fw, err := writer.CreateFormField("message")
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not marshall body: %e", err)})
			return
		}
		_, err = io.Copy(fw, strings.NewReader(message))
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not marshall body: %e", err)})
			return
		}
		part, err := writer.CreateFormFile("filedata", "test.png")
		if err != nil {
			cancel()
		}
		part.Write(buffer.Bytes())
		//log.Printf("bytes to send %v", buffer)
		//writer.WriteField("payload", string(results))
		writer.Close()
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/image", a.mlHost), bodyReader)
		if err != nil {
			cancel()
		}
		tr := &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    600 * time.Second,
			DisableCompression: true,
		}
		client := &http.Client{Transport: tr}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		log.Printf("Send %v", fmt.Sprintf("%s/image", a.mlHost))
		resp, err := client.Do(req)
		//TODO: Handle errors from ml server
		if err != nil {
			cancel()
		}
		defer resp.Body.Close()
		log.Printf("Recieved %v", fmt.Sprintf("%s/image", a.mlHost))
		// Read the processed image into a byte slice
		processedImage, err := io.ReadAll(resp.Body)
		//log.Printf("Processed %v", processedImage)
		if err != nil {
			cancel()
		}
		process <- processedImage
	}(&buffer, requestBody.Message, process)
	select {
	case result = <-process:
		// Return the processed image
		c.Data(http.StatusOK, "image/png", result)
	case <-ctx.Done():
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("timeout from server: %s", a.mlHost)})
	}
}

func (a *AppHandler) SendMessageToML(c *gin.Context) {
	var requestBody PostImageBody
	var result []byte
	if err := c.BindJSON(&requestBody); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no user in query"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second*60*60)
	defer func() {
		recovered := recover()
		if recovered != nil {
			debug.PrintStack()
			err := fmt.Errorf("recovered from: %s", recovered)
			fmt.Println(err)
		}
	}()
	defer cancel()
	process := make(chan []byte)
	go func(message string, process chan []byte) {
		// Prepare a new HTTP POST request to the processing server
		log.Print("Herer")
		results, err := json.Marshal(&PostMlBody{
			Message: message,
		})
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not marshall body: %e", err)})
			return
		}
		bodyReader := bytes.NewReader(results)
		log.Printf("send a request to %v", fmt.Sprintf("%s/message", a.mlHost))
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/message", a.mlHost), bodyReader)
		if err != nil {
			cancel()
		}
		tr := &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    600 * time.Second,
			DisableCompression: true,
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Do(req)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not execute request body: %e", err)})
			return
		}
		log.Printf("analysisis a request to %v", a.mlHost)
		defer resp.Body.Close()
		// Read the processed image into a byte slice
		processedImage, err := io.ReadAll(resp.Body)
		if err != nil {
			cancel()
		}
		log.Printf("recieve %v", processedImage)
		process <- processedImage
	}(requestBody.Message, process)
	log.Print("Waiting")
	select {
	case result = <-process:
		// Return the processed image
		c.Data(http.StatusOK, "image/png", result)
	case <-ctx.Done():
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("timeout from server: %s", a.mlHost)})
	}
}

func (a *AppHandler) SendTrackToML(c *gin.Context) {
	var requestBody PostTrackBody
	var result []byte
	if err := c.BindJSON(&requestBody); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "no user in query"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second*60*60)
	defer func() {
		recovered := recover()
		if recovered != nil {
			debug.PrintStack()
			err := fmt.Errorf("recovered from: %s", recovered)
			fmt.Println(err)
		}
	}()
	defer cancel()
	process := make(chan []byte)
	go func(message string, process chan []byte) {
		// Prepare a new HTTP POST request to the processing server
		log.Print("Herer")
		results, err := json.Marshal(&PostMlBody{
			Message: message,
		})
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not marshall body: %e", err)})
			return
		}
		bodyReader := bytes.NewReader(results)
		log.Printf("send a request to %v", fmt.Sprintf("%s/track", a.mlHostAudio))
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/track", a.mlHostAudio), bodyReader)
		if err != nil {
			cancel()
		}
		tr := &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    600 * time.Second,
			DisableCompression: true,
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Do(req)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not execute request body: %e", err)})
			return
		}
		log.Printf("analysisis a request to %v", a.mlHostAudio)
		defer resp.Body.Close()
		// Read the processed image into a byte slice
		processedImage, err := io.ReadAll(resp.Body)
		if err != nil {
			cancel()
		}
		log.Printf("recieve %v", processedImage)
		process <- processedImage
	}(requestBody.Message, process)
	log.Print("Waiting")
	select {
	case result = <-process:
		// Return the processed image
		c.Data(http.StatusOK, "audio/wav", result)
	case <-ctx.Done():
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("timeout from server: %s", a.mlHostAudio)})
	}
}

func (a *AppHandler) SendMelodyToML(c *gin.Context) {
	log.Print("MAin Herer")
	requestBody := PostTrackBody{Message: c.PostForm("message")}
	var result []byte
	log.Printf("MAin Herer, %v", c.PostForm("message"))

	// Parse the form data, including the uploaded file
	file, _, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()
	//log.Printf("recieve file to send %v and %v", file, loc)
	// Read the file into a buffer
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second*60*60)
	defer func() {
		recovered := recover()
		if recovered != nil {
			debug.PrintStack()
			err := fmt.Errorf("recovered from: %s", recovered)
			fmt.Println(err)
		}
	}()
	defer cancel()
	process := make(chan []byte)
	go func(buffer *bytes.Buffer, message string, process chan []byte) {
		// Prepare a new HTTP POST request to the processing server
		log.Print("Herer")

		bodyReader := new(bytes.Buffer)
		writer := multipart.NewWriter(bodyReader)
		fw, err := writer.CreateFormField("message")
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not marshall body: %e", err)})
			return
		}
		_, err = io.Copy(fw, strings.NewReader(message))
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not marshall body: %e", err)})
			return
		}
		part, err := writer.CreateFormFile("filedata", "test.wav")
		if err != nil {
			cancel()
		}
		part.Write(buffer.Bytes())
		//log.Printf("bytes to send %v", buffer)
		//writer.WriteField("payload", string(results))
		writer.Close()
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/melody", a.mlHostAudio), bodyReader)
		if err != nil {
			cancel()
		}
		tr := &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    600 * time.Second,
			DisableCompression: true,
		}
		client := &http.Client{Transport: tr}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		log.Printf("Send %v", fmt.Sprintf("%s/melody", a.mlHostAudio))
		resp, err := client.Do(req)
		//TODO: Handle errors from ml server
		if err != nil {
			cancel()
		}
		defer resp.Body.Close()
		log.Printf("Recieved %v", fmt.Sprintf("%s/melody", a.mlHostAudio))
		// Read the processed image into a byte slice
		processedImage, err := io.ReadAll(resp.Body)
		//log.Printf("Processed %v", processedImage)
		if err != nil {
			cancel()
		}
		process <- processedImage
	}(&buffer, requestBody.Message, process)
	select {
	case result = <-process:
		// Return the processed image
		c.Data(http.StatusOK, "audio/wav", result)
	case <-ctx.Done():
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("timeout from server: %s", a.mlHostAudio)})
	}
}

func (a *AppHandler) SendTSToML(c *gin.Context) {
	log.Print("MAin Herer")
	predictor := c.PostForm("predictor")
	target := c.PostForm("target")
	log.Printf("MAin Herer, %v", c.PostForm("message"))
	// Parse the form data, including the uploaded file
	file, _, err := c.Request.FormFile("ts")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()
	//log.Printf("recieve file to send %v and %v", file, loc)
	// Read the file into a buffer
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second*60*60)
	defer func() {
		recovered := recover()
		if recovered != nil {
			debug.PrintStack()
			err := fmt.Errorf("recovered from: %s", recovered)
			fmt.Println(err)
		}
	}()
	defer cancel()
	process := make(chan TSResponseBody)
	go func(buffer *bytes.Buffer, predictor, target string, process chan TSResponseBody) {
		// Prepare a new HTTP POST request to the processing server
		log.Print("Herer")

		bodyReader := new(bytes.Buffer)
		writer := multipart.NewWriter(bodyReader)
		fw, err := writer.CreateFormField("predictor")
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not marshall body: %e", err)})
			return
		}
		_, err = io.Copy(fw, strings.NewReader(predictor))
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not marshall body: %e", err)})
			return
		}
		fw, err = writer.CreateFormField("target")
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not marshall body: %e", err)})
			return
		}
		_, err = io.Copy(fw, strings.NewReader(target))
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError,
				gin.H{"message": fmt.Sprintf("can not marshall body: %e", err)})
			return
		}
		part, err := writer.CreateFormFile("filedata", "test.csv")
		if err != nil {
			cancel()
		}
		part.Write(buffer.Bytes())
		//log.Printf("bytes to send %v", buffer)
		//writer.WriteField("payload", string(results))
		writer.Close()
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/ts", a.mlHostTS), bodyReader)
		if err != nil {
			cancel()
		}
		tr := &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    600 * time.Second,
			DisableCompression: true,
		}
		client := &http.Client{Transport: tr}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		log.Printf("Send %v", fmt.Sprintf("%s/melody", a.mlHostAudio))
		resp, err := client.Do(req)
		//TODO: Handle errors from ml server
		if err != nil {
			cancel()
		}
		defer resp.Body.Close()
		log.Printf("Recieved %v", fmt.Sprintf("%s/ts", a.mlHostTS))
		// Read the processed image into a byte slice
		processedBytes, err := io.ReadAll(resp.Body)
		//log.Printf("Processed %v", processedImage)
		if err != nil {
			cancel()
		}
		ts := TSResponseBody{}
		err = json.Unmarshal(processedBytes, &ts)
		if err != nil {
			cancel()
		}
		process <- ts
	}(&buffer, predictor, target, process)

	select {
	case result := <-process:
		// Return the processed image
		c.JSON(http.StatusOK, gin.H{"status": "success", "arrays": result})
	case <-ctx.Done():
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("timeout from server: %s", a.mlHostTS)})
	}

}

func (a *AppHandler) DeleteImage(c *gin.Context) {
	var requestBody DeleteImageBody
	log.Printf("Herer")
	if err := c.BindJSON(&requestBody); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("cannot delete: %e", err)})
		return
	}

	if err := a.s3.DeleteFile(fmt.Sprintf("%s/%s", requestBody.User, requestBody.Name)); err != nil {
		c.IndentedJSON(http.StatusInternalServerError,
			gin.H{"message": fmt.Sprintf("can not delete image from s3: %e", err)})

	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (a *AppHandler) authorize(c *gin.Context) bool {
	// Make sure the user is authenticated. Note that in a production application, we would validate the token signature,
	// make sure it wasn't expired, and attempt to refresh it if it were
	cookie, err := c.Cookie(a.AccessTokenCookieName)
	log.Printf("cookie is %v", cookie)
	if err != nil || cookie == "" || !a.dataStore.VerifyUser(cookie) {
		return false
	}
	return true

}
