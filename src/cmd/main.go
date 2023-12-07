package main

import (
    server "goserv/src/server"
	//"fmt"
	//"log"
	"context"
	cfg "goserv/src/configuration"
    
)

func main() {
	config := cfg.ReadProperties()
    mainContext := context.Background()
	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	_, cancel := context.WithCancel(mainContext)
	defer cancel()
	server.RunServer(config)
	
}