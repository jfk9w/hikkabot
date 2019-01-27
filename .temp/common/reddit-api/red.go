package red

import (
	. "net/http"
	. "sync"
	. "time"

	"strconv"

	"github.com/jfk9w-go/hikkabot/common/httpx"
	"github.com/jfk9w-go/hikkabot/common/logx"
)

const (
	TokenEndpoint = "https://www.reddit.com/api/v1/access_token"
	Endpoint      = "https://oauth.reddit.com"
)

type API struct {
	httpx.HTTP
	Mutex
	Config

	token string
}

func Configure(config Config) *API {
	var (
		http = httpx.Configure(config.HTTP)
		api  = &API{
			HTTP:   http,
			Config: config,
		}
		err = api.RefreshToken()
	)

	if err != nil {
		panic(err)
	}

	go func() {
		var ticker = NewTicker(config.RefreshTokenTimeout.Duration())
		for range ticker.C {
			api.RefreshToken()
		}
	}()

	return api
}

func (api *API) RefreshToken() error {
	log.Debugf("Refreshing access token")

	var req, err = NewRequest(MethodPost, TokenEndpoint, nil)
	if err != nil {
		return err
	}

	req.URL.RawQuery = httpx.Params{
		"grant_type": {"password"},
		"username":   {api.Username},
		"password":   {api.Password},
	}.Encode()

	req.SetBasicAuth(api.ClientID, api.ClientSecret)
	api.setUserAgent(req)

	var resp *Response
	resp, err = api.Do(req)
	if err != nil {
		return err
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}

	err = httpx.ReadToJSON(resp, &token)
	if err != nil {
		return err
	}

	api.Lock()
	api.token = token.AccessToken
	api.Unlock()

	log.Debugf("Received access token: %s", token.AccessToken)

	return nil
}

// path is /r/me_irl/new, for example
func (api *API) Listing(path string, limit int) ([]ThingData, error) {
	var req, err = NewRequest(MethodGet, Endpoint+path, nil)
	if err != nil {
		return nil, err
	}

	api.setUserAgent(req)
	api.setAuthToken(req)

	if limit > 0 {
		req.URL.RawQuery = httpx.Params{}.Set("limit", strconv.Itoa(limit)).Encode()
	}

	var resp *Response
	resp, err = api.Do(req)
	if err != nil {
		return nil, err
	}

	var listing struct {
		Data struct {
			Children []struct {
				Data ThingData `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	err = httpx.ReadToJSON(resp, &listing)
	if err != nil {
		return nil, err
	}

	var things = make([]ThingData, len(listing.Data.Children))
	for i, thing := range listing.Data.Children {
		things[i] = thing.Data
	}

	return things, nil
}

func (api *API) setUserAgent(req *Request) {
	req.Header.Set("User-Agent", api.UserAgent)
}

func (api *API) setAuthToken(req *Request) {
	api.Lock()
	var token = api.token
	api.Unlock()

	req.Header.Set("Authorization", "Bearer "+token)
}

var log = logx.Get("red")
