package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	app "goserv/src/app"
	cfg "goserv/src/configuration"
	db "goserv/src/repository"
	"io"
	"log"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type (
	AuthHandler struct {
		oidcProvider           *oidc.Provider
		dataStore              db.AuthDB
		AuthConfig             *oauth2.Config
		URL                    string
		ClientID               string
		AccessTokenCookieName  string
		RefreshTokenCookieName string
		IDTokenCookieName      string
		oauthStateString       string
	}
)

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func NewAuthHandler(config *cfg.Properties) *AuthHandler {
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

		return &AuthHandler{
			dataStore:              dataConnect,
			oidcProvider:           provider,
			AuthConfig:             authConfig,
			URL:                    config.Server.Name,
			ClientID:               config.Auth.ID,
			AccessTokenCookieName:  config.Auth.AccessTokenCookieName,
			RefreshTokenCookieName: config.Auth.RefreshTokenCookieName,
			IDTokenCookieName:      config.Auth.IDTokenCookieName,
		}
	}
	return &AuthHandler{}
}

func (a *AuthHandler) GetHealth(c *gin.Context) {
	// Now get the ID token so we can show the user's email address

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (a *AuthHandler) Root(c *gin.Context) {
	if !a.authorize(c) {
		c.IndentedJSON(http.StatusNonAuthoritativeInfo, gin.H{"message": "No Authorize to get resourse"})
		return
	}
}

func (a *AuthHandler) Login(c *gin.Context) {
	a.oauthStateString, _ = randString(16)
	// Now get the ID token so we can show the user's email address
	callback, _ := c.Cookie("callback")
	log.Printf("Generated callback %v", callback)
	c.JSON(http.StatusOK, gin.H{"ref": a.AuthConfig.AuthCodeURL(a.oauthStateString)})
}

func (a *AuthHandler) Singin(c *gin.Context) {
	a.oauthStateString, _ = randString(16)
	c.Redirect(http.StatusFound, a.AuthConfig.AuthCodeURL(a.oauthStateString))
}

func (a *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie(a.AccessTokenCookieName, "", int(-1), "/", a.URL, false, false)
	c.SetCookie(a.RefreshTokenCookieName, "", int(-1), "/", a.URL, false, false)
	c.SetCookie(a.IDTokenCookieName, "", int(-1), "/", a.URL, false, false)
}

func (a *AuthHandler) Callback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")
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

	// Write access, refresh, and id tokens to http-only cookies
	c.SetCookie(a.AccessTokenCookieName, token.AccessToken, int(3600), "/", a.URL, false, false)
	c.SetCookie(a.RefreshTokenCookieName, token.RefreshToken, int(3600), "/", a.URL, false, false)
	c.SetCookie(a.IDTokenCookieName, rawIDToken, int(3600), "/", a.URL, false, false)
	a.dataStore.UploadUser(token.AccessToken, token.RefreshToken)
	cookieCallback, err := c.Cookie("callback")
	if err == nil {
		log.Println(cookieCallback)
	} else {
		log.Printf("No cookie %e found", err)
	}
	c.Redirect(http.StatusFound, cookieCallback)
}

func (a *AuthHandler) Account(c *gin.Context) {
	if !a.authorize(c) {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "No Authorize to get resourse"})
		return
	}
	// Now get the ID token so we can show the user's email address
	cookie, err := c.Cookie(a.IDTokenCookieName)

	if err != nil || cookie == "" {
		c.IndentedJSON(
			http.StatusNonAuthoritativeInfo,
			gin.H{"message": "No ID token found"})
		return
	}

	var verifier = a.oidcProvider.Verifier(&oidc.Config{ClientID: a.ClientID})

	idToken, err := verifier.Verify(oauth2.NoContext, cookie)

	if err != nil {
		c.IndentedJSON(
			http.StatusNonAuthoritativeInfo,
			gin.H{"message": "Error verifying ID token: " + err.Error()})
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
		c.IndentedJSON(
			http.StatusNonAuthoritativeInfo,
			gin.H{"message": "Can not parse claims verifying ID token: " + err.Error()})
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
}

func (a *AuthHandler) authorize(c *gin.Context) bool {
	// Make sure the user is authenticated. Note that in a production application,
	// we would validate the token signature,
	// make sure it wasn't expired, and attempt to refresh it if it were
	cookie, err := c.Cookie(a.AccessTokenCookieName)
	if err != nil || cookie == "" || !a.dataStore.VerifyUser(cookie) {
		return false
	}
	return true
}
