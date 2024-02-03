package huawei

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"push-sender/internal/push"

	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"
)

type HmsTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	Scope            string `json:"scope,omitempty"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func (sender *HmsSender) RefreshToken(opts *HmsMessageOpts) (string, error) {
	postData := fmt.Sprintf("grant_type=client_credentials&client_secret=%s&client_id=%s", opts.ApiKey, opts.ClientId)

	request, err := http.NewRequest(http.MethodPost, config.GetString("hms_oauth_api"), bytes.NewBuffer([]byte(postData)))

	if err != nil {
		log.Errorf("newpusher hms: cannot make request %s", err)
		return "", push.ErrorRequest
	}
	request.Header.Add("Host", config.GetString("hms_oauth_host"))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{Transport: transport}

	resp, err := client.Do(request)

	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if err != nil {
		log.Errorf("newpusher hms: cannot send request %s", err)
		return "", push.ErrorRequest
	}

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Errorf("newpusher hms: cannot status request %s [%s]", resp.Status, string(body))
		return "", push.ErrorRequest
	}

	var token HmsTokenResponse

	err = json.Unmarshal(body, &token)

	if err != nil || token.AccessToken == "" {
		log.Errorf("newpusher hms: cannot unmarshal resp request %s", body)
		return "", push.ErrorRequest
	}

	sender.Tokens[opts.ClientId] = token.AccessToken

	return token.AccessToken, nil
}

type HmsSender struct {
	Tokens map[string]string
}

func New() *HmsSender {
	return &HmsSender{
		Tokens: make(map[string]string),
	}
}

func (sender *HmsSender) GetAuthToken(opts *HmsMessageOpts) (string, error) {
	if token, ok := sender.Tokens[opts.ClientId]; ok {
		return token, nil
	}
	return sender.RefreshToken(opts)
}
