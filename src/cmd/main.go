package main

import (
	cfg "goserv/src/configuration"
	server "goserv/src/server"
)

func main() {
	config := cfg.ReadProperties()
	server.RunServer(config)
}
