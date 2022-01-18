package main

import (
	"blossom/web"
	"github.com/rs/zerolog"
)

//goal of the app is to provide insight & functionality for a single user's portfolio
func main() {
	//setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	//logger = log.With().Str("component", "Main").Logger()

	//start API on localhost:8080
	web.Start()
}

