package main

import (
	"github_apis/api"
	"log"
)

func main() {
	public_apis_api := api.New("https", "public-apis-api.herokuapp.com")
	catsCh := public_apis_api.GetApisFromCategory("Animals")
	for {
		cats, err := (<-catsCh).Unwrap()
		if err != nil {
			if err == api.NoMoreResponse {
				break
			}
			panic(err)
		}
		log.Println(cats)
	}
	println("Safely exited")
}
