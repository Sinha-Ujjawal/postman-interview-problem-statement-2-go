package main

import (
	"github_apis/api"
	"log"
)

func main() {
	public_apis_api := api.New(
		"https",
		"public-apis-api.herokuapp.com",
		api.ApiEndpoints{
			Auth:       api.Endpoint{Path: "/api/v1/auth/token"},
			Categories: api.Endpoint{Path: "/api/v1/apis/categories"},
		},
	)
	catsCh := public_apis_api.GetCategories()
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
