package android

/**
 *	Implemented old fcm push messaging HTTP Proto
 */

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/goccy/go-json"

	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"
	"golang.org/x/oauth2/google"

	"push-sender/internal/containers/maps"
	"push-sender/internal/push"
)

const (
	firebaseScope       = "https://www.googleapis.com/auth/firebase.messaging"
	apnsAuthError       = "APNS_AUTH_ERROR"
	internalError       = "INTERNAL"
	thirdPartyAuthError = "THIRD_PARTY_AUTH_ERROR"
	invalidArgument     = "INVALID_ARGUMENT"
	quotaExceeded       = "QUOTA_EXCEEDED"
	senderIDMismatch    = "SENDER_ID_MISMATCH"
	unregistered        = "UNREGISTERED"
	unavailable         = "UNAVAILABLE"
	unauth              = "UNAUTHENTICATED"
)

var transport *http.Transport

func init() {
	config.SetDefault("fcm_send_api_v1", "https://fcm.googleapis.com/v1/projects/%s/messages:send")
	config.SetDefault("retry_push", 2)
	transport = &http.Transport{
		MaxIdleConnsPerHost: 10,
		MaxIdleConns:        100,
		IdleConnTimeout:     1 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
}

func MakeFcmMessageOpts(jwtString string, timeToLife int) (*FcmMessageOpts, error) {
	cfg, err := google.JWTConfigFromJSON([]byte(jwtString), firebaseScope)

	if err != nil {
		return nil, fmt.Errorf("bad jwt data %w", err)
	}

	type jwtJson struct {
		ProjectID string `json:"project_id"`
	}

	var f jwtJson
	if err := json.Unmarshal([]byte(jwtString), &f); err != nil {
		return nil, err
	}

	if f.ProjectID == "" {
		return nil, fmt.Errorf("bad jwt data %w", err)
	}

	opts := &FcmMessageOpts{
		ProjectId:  f.ProjectID,
		Cfg:        cfg,
		TimeToLive: timeToLife,
	}

	return opts, nil
}

func New() *AndroidSender {
	return &AndroidSender{
		Tokens: make(map[string]string),
	}
}

func (opts *FcmMessageOpts) Valid() bool {
	return opts.Cfg != nil
}

func parseReply(body []byte) error {
	var resp FcmResponse

	err := json.Unmarshal(body, &resp)

	if err != nil {
		log.Errorf("newpusher android: Parse reply error %s %s", err, string(body))
		return push.ErrorServiceUnavailable
	}

	if resp.Code > 0 {
		log.Errorf("newpusher android: bad reply %s", string(body))
		switch resp.Status {
		case unregistered:
			return push.ErrorTokenRemoved
		case unauth:
			return push.ErrorRefreshToken
		default:
			return push.ErrorServiceUnavailable
		}
	}

	fmt.Println(string(body))

	return nil
}

func (sender *AndroidSender) GetToken(opts *FcmMessageOpts, refreshToken bool) (string, error) {
	if maps.Exists(sender.Tokens, opts.ProjectId) && !refreshToken {
		token, _ := sender.Tokens[opts.ProjectId]
		return token, nil
	}

	ts := opts.Cfg.TokenSource(context.Background())

	token, err := ts.Token()

	if err != nil {
		log.Errorf(" %s", err)
		return "", push.ErrorInvalidKey
	}

	sender.Tokens[opts.ProjectId] = token.AccessToken

	return token.AccessToken, nil
}

func (sender *AndroidSender) Send(to string, data map[string]string, opts *FcmMessageOpts) error {
	if opts == nil || !opts.Valid() || to == "" {
		log.Errorf("newpusher android: opts not valid [%#v] [%s]", opts, to)
		return push.ErrorRequest
	}

	fcmMsg := FcmMessageProto{}
	fcmMsg.Message.Data = data
	fcmMsg.Message.Token = to

	j, err := json.Marshal(&fcmMsg)

	if err != nil {
		log.Errorf("newpusher android: cannot j, err := json.Marshal(&msg) %s", err)
		return push.ErrorRequest
	}

	refreshToken := false

	for i := 0; i < config.GetInt("retry_push"); i++ {
		var request *http.Request
		request, err = http.NewRequest(http.MethodPost, fmt.Sprintf(config.GetString("fcm_send_api_v1"), opts.ProjectId), bytes.NewBuffer(j))

		if err != nil {
			log.Errorf("newpusher android: cannot read options %s", err)
			return push.ErrorRequest
		}

		apiKey, err := sender.GetToken(opts, refreshToken)

		if err != nil {
			log.Errorf("new pusher cannot get push token %s", err)
			return err
		}

		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))
		request.Header.Add("Content-Type", "application/json")

		client := http.Client{Transport: transport}

		resp, err := client.Do(request)

		defer func() {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}()

		if err != nil {
			log.Errorf("newpusher android: cannot send request [%#v] %s", fcmMsg, err)
			return push.ErrorRequest
		}

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			log.Errorf("newpusher android: cannot read data %s", err)
			return push.ErrorServiceUnavailable
		}

		switch resp.StatusCode {
		case 401:
			if err = parseReply(body); errors.Is(err, push.ErrorRefreshToken) {
				refreshToken = true
				continue
			} else {
				log.Errorf("newpusher android: ErrorInvalidKey %s %s [%#v]", resp.Status, string(body), fcmMsg)
				return err
			}
		case 400, 500:
			log.Errorf("newpusher android: ServiceUnavailable %s %s [%#v]", resp.Status, string(body), fcmMsg)
			return push.ErrorServiceUnavailable
		}

		err = parseReply(body)

		if err != nil && errors.Is(err, push.ErrorTokenRemoved) {
			return err
		}
		if err != nil {
			refreshToken = true
		} else {
			return nil
		}

	}

	return err
}

/**
response
*/
