package main

import (
	"fmt"
	"github_apis/api"
)

func main() {
	public_apis_api := api.New(
		"https",
		"public-apis-api.herokuapp.com",
		// api.WithLogger(log.Default()),
	)
	catsCh := public_apis_api.GetApisFromCategory("Animals")
	for {
		cats, err := (<-catsCh).Unwrap()
		if err != nil {
			if err == api.NoMoreResponse {
				break
			}
			panic(err)
		}
		fmt.Println(cats)
	}
	println("Safely exited")
}
