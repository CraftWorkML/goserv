package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {

	url := "https://gitlab.aramcoinnovations.com/.well-known/openid-configuration"

	req, _ := http.NewRequest("GET", url, nil)

	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		IdleConnTimeout:    100000 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	res, err := client.Do(req)

	if err != nil {
		fmt.Printf("error:%e\n", err)
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	fmt.Println(res)
	fmt.Println(string(body))

}
