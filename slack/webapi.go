package slack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const SlackWebAPIEndpoint = "https://slack.com/api"

type WebAPIError struct {
	Method, URL, Response string
}

func (e *WebAPIError) Error() string {
	return fmt.Sprintf("%s (method: %s, url: %s)", e.Response, e.Method, e.URL)
}

type WebAPI struct {
	c     *http.Client
	token string

	BaseURL string
}

func NewWebAPI(token string, httpClient *http.Client) *WebAPI {
	api := &WebAPI{
		token:   token,
		c:       httpClient,
		BaseURL: SlackWebAPIEndpoint,
	}

	if api.c == nil {
		api.c = http.DefaultClient
	}

	return api
}

func (api *WebAPI) SetChannelTopic(channelID, topic string) error {
	const method = "channels.setTopic"

	params := url.Values{}
	params.Add("channel", channelID)
	params.Add("topic", topic)

	_, _, err := api.Call(method, params)
	return err
}

func (api *WebAPI) GetChannelTopic(channelID string) (string, error) {
	const method = "channels.info"

	params := url.Values{}
	params.Add("channel", channelID)

	resp, requestURL, err := api.Call(method, params)
	if err != nil {
		return "", err
	}

	var v struct {
		Channel struct {
			Topic struct {
				Value string `json:"value"`
			} `json:"topic"`
		} `json:"channel"`
	}
	if err := json.Unmarshal(resp, &v); err != nil {
		return "", wrapError(fmt.Errorf("failed to decode response body %q (%s)", resp, err), method, requestURL)
	}

	return v.Channel.Topic.Value, nil
}

func (api *WebAPI) Call(method string, params url.Values) (response []byte, u *url.URL, err error) {
	req, err := http.NewRequest("GET", api.BaseURL+"/"+method, nil)
	if err != nil {
		return nil, &url.URL{Opaque: api.BaseURL + "/" + method}, wrapError(fmt.Errorf("failed to build WebAPI request (%s)", err), method, nil)
	}

	if params == nil {
		params = url.Values{}
	}

	params.Add("token", api.token)
	req.URL.RawQuery = params.Encode()

	resp, err := api.c.Do(req)
	if err != nil {
		return nil, req.URL, wrapError(fmt.Errorf("failed to call method (%s)", err), method, req.URL)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, req.URL, wrapError(fmt.Errorf("failed to read response body (%s)", err), method, req.URL)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, req.URL, wrapError(fmt.Errorf("WebAPI responded with HTTP %d %q", resp.StatusCode, body), method, req.URL)
	}

	var v struct {
		Ok    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, req.URL, wrapError(fmt.Errorf("failed to decode response body %q (%s)", body, err), method, req.URL)
	}

	if !v.Ok {
		if v.Error != "" {
			return nil, req.URL, wrapError(fmt.Errorf("WebAPI returned error (%s)", v.Error), method, req.URL)
		} else {
			return nil, req.URL, wrapError(errors.New("WebAPI returned unknown error"), method, req.URL)
		}
	}

	return body, req.URL, nil
}

func wrapError(err error, method string, url *url.URL) *WebAPIError {
	e := &WebAPIError{
		Method:   method,
		URL:      url.String(),
		Response: err.Error(),
	}

	if url != nil {
		url.Query().Set("token", "[hidden]")
		e.URL = url.String()
	}

	return e
}
