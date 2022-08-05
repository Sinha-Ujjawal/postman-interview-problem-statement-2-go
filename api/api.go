package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github_apis/result"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const DefaultMaxAttempts = 10

var NoMoreResponse = errors.New("No More Response!")
var DefaultAuthEndpoint = Endpoint{Path: "/api/v1/auth/token"}
var DefaultCategoriesEndpoint = Endpoint{Path: "/api/v1/apis/categories"}
var DefaultEntryEndpoint = Endpoint{Path: "api/v1/apis/entry"}

type Endpoint struct {
	Path string
}

type apiEndpoints struct {
	auth       Endpoint
	categories Endpoint
	entry      Endpoint
}

type api struct {
	scheme      string
	host        string
	endpoints   apiEndpoints
	authToken   string
	maxAttempts uint8
}

type apiOption func(*api)

func WithMaxAttempts(maxAttempts uint8) apiOption {
	return func(api *api) {
		api.maxAttempts = maxAttempts
	}
}

func WithAuthEndpoint(endpoint Endpoint) apiOption {
	return func(api *api) {
		api.endpoints.auth = endpoint
	}
}

func WithCategoriesEndpoint(endpoint Endpoint) apiOption {
	return func(api *api) {
		api.endpoints.categories = endpoint
	}
}

func WithEntryEndpoint(endpoint Endpoint) apiOption {
	return func(api *api) {
		api.endpoints.entry = endpoint
	}
}

func New(
	scheme string,
	host string,
	opts ...apiOption,
) api {
	a := api{
		scheme: scheme,
		host:   host,
		endpoints: apiEndpoints{
			auth:       DefaultAuthEndpoint,
			categories: DefaultCategoriesEndpoint,
			entry:      DefaultEntryEndpoint,
		},
		maxAttempts: DefaultMaxAttempts,
	}
	for _, opt := range opts {
		opt(&a)
	}
	return a
}

func bearerToken(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}

func (a *api) get(u *url.URL) result.Result[[]byte] {
	ustr := u.String()
	log.Printf("Get Request: %s\n", ustr)
	attempts := uint8(0)
	t := 1
	client := http.DefaultClient
	for {
		if attempts >= a.maxAttempts {
			break
		}
		req, err := http.NewRequest("GET", ustr, nil)
		if err != nil {
			return result.Err[[]byte](err)
		}
		req.Header.Set("Authorization", bearerToken(a.authToken))
		resp, err := client.Do(req)
		if err != nil {
			return result.Err[[]byte](err)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			log.Printf(
				"Max Request Made! Total attempts: %d made out of %d\n",
				attempts,
				a.maxAttempts,
			)
			tts := t
			time.Sleep(time.Duration(tts) * time.Second)
			attempts += 1
			t <<= 1
		} else if resp.StatusCode == http.StatusOK {
			log.Println("Status OK, returning response")
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return result.Err[[]byte](err)
			}
			return result.Ok(body)
		} else {
			resp.Body.Close()
			log.Println("Unauthorized or token expired, reauthentication...")
			err = a.setToken()
			if err != nil {
				return result.Err[[]byte](err)
			}
		}
	}
	return result.Err[[]byte](errors.New(fmt.Sprintf("Max attempts: %d reached!", a.maxAttempts)))
}

func getPagedResponse[T any](
	u *url.URL,
	a *api,
	payloadConverter func([]byte) T,
	cout chan<- T,
) {
	setPage := func(u *url.URL, page uint32) {
		rq := u.Query()
		rq.Set("page", fmt.Sprintf("%d", page))
		u.RawQuery = rq.Encode()
	}
	page := uint32(1)
	for {
		setPage(u, page)
		payload, err := a.get(u).Unwrap()
		if err != nil {
			break
		}
		cout <- payloadConverter(payload)
		page += 1
	}
}

type tokenResponse struct {
	Token string `json:"token"`
}

func (a *api) setToken() error {
	u := &url.URL{
		Scheme: a.scheme,
		Host:   a.host,
		Path:   a.endpoints.auth.Path,
	}
	body, err := a.get(u).Unwrap()
	if err != nil {
		return err
	}
	var t tokenResponse
	err = json.Unmarshal(body, &t)
	if err != nil {
		return err
	}
	a.authToken = t.Token
	return nil
}

func (a api) categoriesURL() *url.URL {
	return &url.URL{
		Scheme: a.scheme,
		Host:   a.host,
		Path:   a.endpoints.categories.Path,
	}
}

func (a *api) GetCategories() <-chan result.Result[[]string] {
	ret := make(chan result.Result[[]string])

	type categories struct {
		Categories []string `json:"categories"`
	}

	payloadConverter := func(data []byte) result.Result[[]string] {
		var cats categories
		err := json.Unmarshal(data, &cats)
		if err != nil {
			return result.Err[[]string](err)
		} else {
			if len(cats.Categories) == 0 {
				return result.Err[[]string](NoMoreResponse)
			}
			return result.Ok(cats.Categories)
		}
	}

	go func() {
		getPagedResponse(a.categoriesURL(), a, payloadConverter, ret)
	}()

	return ret
}

type categoryApi struct {
	category string
	api      string
}

func (a api) entryURL(category string) *url.URL {
	u := &url.URL{
		Scheme: a.scheme,
		Host:   a.host,
		Path:   a.endpoints.entry.Path,
	}
	rq := u.Query()
	rq.Set("category", category)
	u.RawQuery = rq.Encode()
	return u
}

func (a *api) GetApisFromCategory(category string) <-chan result.Result[[]categoryApi] {
	ret := make(chan result.Result[[]categoryApi])

	type property struct {
		Link string `json:"Link"`
	}

	type categoryApis struct {
		Properties []property `json:"categories"`
	}

	payloadConverter := func(data []byte) result.Result[[]categoryApi] {
		var resp categoryApis
		err := json.Unmarshal(data, &resp)
		if err != nil {
			return result.Err[[]categoryApi](err)
		} else {
			if len(resp.Properties) == 0 {
				return result.Err[[]categoryApi](NoMoreResponse)
			}
			var ret []categoryApi
			for _, p := range resp.Properties {
				ret = append(ret, categoryApi{category: category, api: p.Link})
			}
			return result.Ok(ret)
		}
	}

	go func() {
		getPagedResponse(a.entryURL(category), a, payloadConverter, ret)
	}()

	return ret
}
