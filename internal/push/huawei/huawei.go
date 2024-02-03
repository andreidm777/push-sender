package huawei

/**
 *	Implemented hms push messaging HTTP Proto
 */

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"push-sender/internal/push"

	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"
)

var transport *http.Transport

func init() {
	config.SetDefault("hms_send_api", "https://159.138.203.26/v1/%s/messages:send")
	config.SetDefault("hms_send_host", "push-api.cloud.huawei.com")
	config.SetDefault("hms_oauth_api", "https://159.138.204.255/oauth2/v3/token")
	config.SetDefault("hms_oauth_host", "oauth-login.cloud.huawei.com")
	transport = &http.Transport{
		MaxIdleConnsPerHost: 10,
		MaxIdleConns:        100,
		IdleConnTimeout:     1 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
}

type InnerMessage struct {
	Data  string   `json:"data"`
	Token []string `json:"token"`
}

type HmsMessage struct {
	ValidateOnly bool         `json:"validate_only"`
	Message      InnerMessage `json:"message"`
}

type HmsMessageOpts struct {
	ApiKey      string
	ClientId    string
	PackageName string
}

type HmsMessageResponse struct {
	Code      string `json:"code"`
	Msg       string `json:"msg"`
	RequestId string `json:"requestId"`
}

func (opts *HmsMessageOpts) Valid() bool {
	return opts.ApiKey != "" && opts.ClientId != ""
}

func (sender *HmsSender) Send(to string, data string, opts *HmsMessageOpts) error {
	if opts == nil || !opts.Valid() || to == "" {
		log.Errorf("newpusher hms: opts not valid [%#v] [%s]", opts, to)
		return push.ErrorRequest
	}

	/* jData, _ := json.Marshal(&data) */

	msg := &HmsMessage{
		Message: InnerMessage{
			Data:  string(data),
			Token: []string{to},
		},
	}

	j, err := json.Marshal(&msg)

	if err != nil {
		log.Errorf("newpusher hms: cannot j, err := json.Marshal(&msg) %s", err)
		return push.ErrorRequest
	}

	for i := 0; i < 2; i++ {
		token, err := sender.GetAuthToken(opts)

		if err != nil {
			log.Errorf("newpusher hms: get auth token %s", err)
			return push.ErrorInvalidKey
		}

		request, err := http.NewRequest(http.MethodPost, fmt.Sprintf(config.GetString("hms_send_api"), opts.ClientId), bytes.NewBuffer(j))

		if err != nil {
			log.Errorf("newpusher hms: cannot read options %s", err)
			return push.ErrorRequest
		}
		request.Header.Add("Host", config.GetString("hms_send_host"))
		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		request.Header.Add("Content-Type", "application/json")

		client := http.Client{Transport: transport}

		resp, err := client.Do(request)

		defer func() {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}()

		if err != nil {
			log.Errorf("newpusher hms: cannot send request %s", err)
			return push.ErrorRequest
		}

		body, _ := io.ReadAll(resp.Body)

		switch resp.StatusCode {
		case 401:
			log.Errorf("newpusher hms: refresh token %s %v", resp.Status, string(body))
			sender.RefreshToken(opts)
			continue
		case 400, 404, 500, 502:
			log.Errorf("newpusher hms: ServiceUnavailable %s %s", resp.Status, string(body))
			return push.ErrorServiceUnavailable
		case 503:
			log.Errorf("newpusher hms: ratelimit %s %s", resp.Status, string(body))
			return push.ErrorRateLimit
		}

		err = parseReply(string(body))

		if errors.Is(err, push.ErrorRefreshToken) {
			sender.RefreshToken(opts)
			continue
		}

		return err
	}

	return push.ErrorServiceUnavailable
}

func parseReply(body string) error {
	var resp HmsMessageResponse
	err := json.Unmarshal([]byte(body), &resp)
	if err != nil {
		log.Errorf("newpusher hms: parse reponse error %s %s", err, body)
		return push.ErrorServiceUnavailable
	}

	switch resp.Code {
	case "80000000":
		return nil
	case "80100000":
		return parseMsg(resp.Msg)
	case "80200001", "80200003":
		log.Infof("newpusher hms: oauth expired %#v", resp)
		return push.ErrorRefreshToken
	case "80300007":
		return push.ErrorTokenRemoved
	default:
		log.Errorf("hms: response error is %#v", resp)
		return fmt.Errorf("hms: error [%w]", push.PushError(resp.Code))
	}
}

func parseMsg(body string) error {
	type HmsMsg struct {
		Success      int      `json:"success"`
		Failure      int      `json:"failure"`
		IllegalToken []string `json:"illegal_tokens,omitempty"`
	}

	var hmsMsg HmsMsg

	err := json.Unmarshal([]byte(body), &hmsMsg)

	if err != nil {
		log.Errorf("hms: parse response msg error %s %s", err, body)
		return push.ErrorServiceUnavailable
	}

	if hmsMsg.Success == 1 {
		return nil
	}

	if hmsMsg.Failure == 1 && len(hmsMsg.IllegalToken) > 0 {
		log.Errorf("hms: token bad %#v", hmsMsg)
		return push.ErrorTokenRemoved
	}
	log.Errorf("hms: strange reply %#v", hmsMsg)
	return push.ErrorServiceUnavailable
}

/**
https://developer.huawei.com/consumer/en/doc/development/HMSCore-References/https-send-api-0000001050986197#section13968115715131

200  Success. -

400 Incorrect parameter. Rectify the fault based on the status code description.

401 Authentication failed. Verify the access token in the Authorization parameter in the request HTTP header.

404 Service not found. Verify that the request URI is correct.

500 Internal service error. Contact technical support.

502 The connection request is abnormal, generally because the network is unstable. Try again later or contact technical support.

503 Traffic control.
Set the average push speed to a value smaller than the QPS quota provided by Huawei. For details about the QPS quota, please refer to FAQs.
Set the average push interval. Do not push messages too frequently in a period of time.

Service Result Code Description Solution

80000000            Success. N/A

80100000   The message is successfully sent to some tokens. Tokens identified by illegal_tokens are those to which the message failed to be sent. Response example:

{
    "code": "80100000",
    "msg": "{\"success\":3,\"failure\":1,\"illegal_tokens\":[\"xxx\"]}",
    "requestId": ""
}
Verify these tokens in the return value.

80100001 Some request parameters are incorrect. Response example:

{
    "code": "80100001",
    "msg": "UnSupported svc",
    "requestId": ""
}

Verify the request parameters as prompted in the response.

80100003 Incorrect message structure.
Verify the parameters in the message structure as prompted in the response.

80100004 The message expiration time is earlier than the current time.
Verify the message field ttl.

80100013 The collapse_key message field is invalid.
Verify the message field collapse_key.

80100017 A maximum of 100 topic-based messages can be sent at the same time.
Increase the interval for sending topic-based messages.

80200001 OAuth authentication error.
The access token in the Authorization parameter in the request HTTP header failed to be authenticated. Ensure that the access token is correct.
The message does not contain the Authorization parameter, or the Authorization parameter is left empty.
The app ID used for applying for the access token is different from that in the message. For example, the access token applied using the ID of app A is used to send messages to app B.

80200003 OAuth token expired.
The access token in the Authorization parameter in the request HTTP header has expired. Obtain a new access token.

80300002
The current app does not have the permission to send messages.

Sign in to AppGallery Connect and verify that Push Kit is enabled.
If HMS Core Push SDK 2.0 is integrated, remove the backslash (\) from the escape character in the access token, then URL-encode the token.
Check whether the token of the user matches that of the app.
If Push Kit functions normally in the Chinese mainland and result code 80300002 is returned only for devices where your app is running outside the Chinese mainland, you need to enable Push Kit for devices outside the Chinese mainland. Find your app from My apps, disable Push Kit, and then enable it again.
NOTE
If you wish to enable Push Kit only, upload the APK of your app to HUAWEI AppGallery first (you can save the APK as a draft). Otherwise, you cannot find the app when enabling Push Kit again.
Check whether there are errors in the body of the message sent.
Test the message push function in AppGallery Connect. If the test is successful, an error occurs when you call the API.
In the multi-sender scenario, check the API prototype.

80300007 All tokens are invalid.
In principle, the tokens of different apps on the same device are different. Actually, the same tokens may exist by mistake.
The package name and app ID configured for the app on the device are different from those obtained in AppGallery Connect.
Check whether the access token URL is correct.
Check whether the message sending URL is correct.
SDK 2.0 URL: https://api.push.hicloud.com/pushsend.do

SDK 3.0+ URL: https://push-api.cloud.huawei.com/v1/[appId]/messages:send

80300008
The message body size (excluding the token) exceeds the default value (4096 bytes).
Reduce the message body size.

80300010
The number of tokens in the message body exceeds the default value.
Reduce the number of tokens and send messages to the tokens in batches.

80300013
Invalid receipt URL.
Verify that your receipt URL is correct and the receipt certificate does not expire.

80600003
Failed to request the OAuth service.
Check the OAuth 2.0 client ID and client secret.

81000001
An internal error of the system occurs.
Contact technical support.
*/
