package main

import (
	"log"
	"xray-checker/config"
)

func main() {
	log.Println("Xray Checker starting...")
	config.Parse("v1.0.0") // Parse CLI arguments and environment variables
	log.Println("Xray Checker started.")
}