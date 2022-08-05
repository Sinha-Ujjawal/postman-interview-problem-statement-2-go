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

const defaultMaxAttempts = 10

var NoMoreResponse = errors.New("No More Response!")

type Endpoint struct {
	Path string
}

type ApiEndpoints struct {
	Auth       Endpoint
	Categories Endpoint
}

type api struct {
	scheme      string
	host        string
	endpoints   ApiEndpoints
	authToken   string
	maxAttempts uint8
}

type apiOpts func(*api)

func WithMaxAttempts(maxAttempts uint8) apiOpts {
	return func(api *api) {
		api.maxAttempts = maxAttempts
	}
}

func New(
	scheme string,
	host string,
	endpoints ApiEndpoints,
	opts ...apiOpts,
) api {
	a := api{scheme: scheme, host: host, endpoints: endpoints, maxAttempts: defaultMaxAttempts}
	for _, opt := range opts {
		opt(&a)
	}
	return a
}

func bearerToken(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}

func get(u *url.URL, a *api) result.Result[[]byte] {
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
		payload, err := get(u, a).Unwrap()
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
	auth := a.endpoints.Auth
	u := &url.URL{
		Scheme: a.scheme,
		Host:   a.host,
		Path:   auth.Path,
	}
	body, err := get(u, a).Unwrap()
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
	categories := a.endpoints.Categories
	return &url.URL{
		Scheme: a.scheme,
		Host:   a.host,
		Path:   categories.Path,
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
