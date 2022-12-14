package main

import (
	"github_apis/api"
	"log"
	"os"
)

const logFilePath = "logs.log"

func main() {
	f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	log.SetOutput(f)
	public_apis_api := api.New(
		"https",
		"public-apis-api.herokuapp.com",
		api.WithLogger(log.Default()),
	)
	totalApis := 0
	for apisErr := range public_apis_api.GetApis() {
		apis, err := apisErr.Unwrap()
		if err != nil {
			panic(err)
		}
		log.Println(apis)
		totalApis += len(apis)
	}
	log.Println("Safely exited")
	log.Printf("Got %d apis\n", totalApis)
}
