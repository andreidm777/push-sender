package rustore

/**
 *	Implemented old fcm push messaging HTTP Proto
 */

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"

	"push-sender/internal/push"
)

var transport *http.Transport

const (
	INVALID_ARGUMENT  = "INVALID_ARGUMENT"  //— неправильно указаны параметры запроса при отправке сообщения.
	INTERNAL          = "INTERNAL"          //— внутренняя ошибка сервиса.
	TOO_MANY_REQUESTS = "TOO_MANY_REQUESTS" //— превышено количество попыток отправить сообщение.
	PERMISSION_DENIED = "PERMISSION_DENIED" //— неправильно указан сервисный ключ.
	NOT_FOUND         = "NOT_FOUND"         //— неправильно указан пуш токен пользователя.
	MAX_SIZE          = 4096
)

func init() {
	config.SetDefault("rustore_send_api", "https://vkpns.rustore.ru/v1/projects/{project_id}/messages:send")
	transport = &http.Transport{
		MaxIdleConnsPerHost: 10,
		MaxIdleConns:        100,
		IdleConnTimeout:     1 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
}

type RuStoreMessage struct {
	Token string            `json:"token"`
	Data  map[string]string `json:"data"`
}

type RuStoreProto struct {
	Message      RuStoreMessage `json:"message"`
	ValidateOnly bool           `json:"validate_only,omitempty"`
}

type RuStoreMessageOpts struct {
	ApiKey    string
	ProjectId string
}

func (opts *RuStoreMessageOpts) Valid() bool {
	return opts.ApiKey != "" && opts.ProjectId != ""
}

type RuStoreResponse struct {
	Error struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Status  push.PushError `json:"status"`
	} `json:"error,omitempty"`
}

func parseReply(body []byte) error {
	var resp RuStoreResponse

	err := json.Unmarshal(body, &resp)

	if err != nil {
		log.Errorf("newpusher rustore: Parse reply error %s %s", err, string(body))
		return push.ErrorServiceUnavailable
	}
	if resp.Error.Code == 0 {
		return nil
	}

	log.Errorf("newpusher rustore: token bad %v", resp)

	switch resp.Error.Status {
	case NOT_FOUND:
		return push.ErrorTokenRemoved
	}

	return fmt.Errorf("error [%w]", resp.Error.Status)
}

func Send(to string, data string, opts *RuStoreMessageOpts) error {
	if opts == nil || !opts.Valid() || to == "" {
		log.Errorf("newpusher rustore: opts not valid [%#v] [%s]", opts, to)
		return push.ErrorRequest
	}

	ruStoreMsg := RuStoreProto{
		Message: RuStoreMessage{
			Token: to,
			Data:  make(map[string]string),
		},
	}

	ruStoreMsg.Message.Data["data"] = data

	j, err := json.Marshal(&ruStoreMsg)

	if err != nil {
		log.Errorf("newpusher rustore: cannot j, err := json.Marshal(&msg) %s", err)
		return push.ErrorRequest
	}

	mUrl := strings.ReplaceAll(config.GetString("rustore_send_api"), "{project_id}", opts.ProjectId)

	request, err := http.NewRequest(http.MethodPost, mUrl, bytes.NewBuffer(j))

	if err != nil {
		log.Errorf("newpusher rustore: cannot read options %s", err)
		return push.ErrorRequest
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", opts.ApiKey))
	request.Header.Add("Content-Type", "application/json")

	client := http.Client{Transport: transport}

	resp, err := client.Do(request)

	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if err != nil {
		log.Errorf("newpusher rustore: cannot send request [%#v] %s", ruStoreMsg, err)
		return push.ErrorRequest
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Errorf("newpusher android: cannot read data %s", err)
		return push.ErrorServiceUnavailable
	}

	return parseReply(body)
}

/**
response
{\"multicast_id\":2891136516237280301,\"success\":1,\"failure\":0,\"canonical_ids\":0,\"results\":[{\"message_id\":\"0:1673874427017057%7b0e42f9f9fd7ecd\"} - good
{\"multicast_id\":5810316827628703959,\"success\":0,\"failure\":1,\"canonical_ids\":0,\"results\":[{\"error\":\"InvalidRegistration\"}]}] - bad pushToken

"401 INVALID_KEY <HTML>\n<HEAD>\n<TITLE>INVALID_KEY</TITLE>\n</HEAD>\n<BODY BGCOLOR=\"#FFFFFF\" TEXT=\"#000000\">\n<H1>INVALID_KEY</H1>\n<H2>Error 401</H2>\n</BODY>\n</HTML>\n" - bad api key

Error	HTTP Code	Recommended Action
Missing Registration Token	200 + error:MissingRegistration	Check that the request contains a registration token (in the registration_id in a plain text message, or in the to or registration_ids field in JSON).
Invalid Registration Token	200 + error:InvalidRegistration	Check the format of the registration token you pass to the server. Make sure it matches the registration token the client app receives from registering with Firebase Notifications. Do not truncate or add additional characters.
Unregistered Device	        200 + error:NotRegistered	An existing registration token may cease to be valid in a number of scenarios, including:
                                 If the client app unregisters with FCM.
                                 If the client app is automatically unregistered, which can happen if the user uninstalls the application. For example, on iOS, if the APNs Feedback Service reported the APNs token as invalid.
                                 If the registration token expires (for example, Google might decide to refresh registration tokens, or the APNs token has expired for iOS devices).
                                 If the client app is updated but the new version is not configured to receive messages.
                                 For all these cases, remove this registration token from the app server and stop using it to send messages.
Invalid Package Name	    200 + error:InvalidPackageName	Make sure the message was addressed to a registration token whose package name matches the value passed in the request.
Authentication Error	    401	The sender account used to send a message couldn't be authenticated. Possible causes are:
                                Authorization header missing or with invalid syntax in HTTP request.
                                The Firebase project that the specified server key belongs to is incorrect.
                                Legacy server keys only—the request originated from a server not whitelisted in the Server key IPs.
                                Check that the token you're sending inside the Authentication header is the correct server key associated with your project. See Checking the validity of a server key for details. If you are using a legacy server key, you're recommended to upgrade to a new key that has no IP restrictions. See Migrate legacy server keys.
Mismatched Sender	        200 + error:MismatchSenderId	A registration token is tied to a certain group of senders. When a client app registers for FCM, it must specify which senders are allowed to send messages. You should use one of those sender IDs when sending messages to the client app. If you switch to a different sender, the existing registration tokens won't work.
Invalid JSON	            400	Check that the JSON message is properly formatted and contains valid fields (for instance, making sure the right data type is passed in).
Invalid Parameters	        400 + error:InvalidParameters	Check that the provided parameters have the right name and type.
Message Too Big	            200 + error:MessageTooBig	Check that the total size of the payload data included in a message does not exceed FCM limits: 4096 bytes for most messages, or 2048 bytes in the case of messages to topics. This includes both the keys and the values.
Invalid Data Key	        200 + error:InvalidDataKey	Check that the payload data does not contain a key (such as from, or gcm, or any value prefixed by google) that is used internally by FCM. Note that some words (such as collapse_key) are also used by FCM but are allowed in the payload, in which case the payload value will be overridden by the FCM value.
Invalid Time to Live	    200 + error:InvalidTtl	Check that the value used in time_to_live is an integer representing a duration in seconds between 0 and 2,419,200 (4 weeks).
Timeout	5xx or              200 + error:Unavailable	The server couldn't process the request in time. Retry the same request, but you must:
													Honor the Retry-After header if it is included in the response from the FCM Connection Server.
													Implement exponential back-off in your retry mechanism. (e.g. if you waited one second before the first retry, wait at least two second before the next one, then 4 seconds and so on). If you're sending multiple messages, delay each one independently by an additional random amount to avoid issuing a new request for all messages at the same time.
													Senders that cause problems risk being blacklisted.

Internal Server Error		500 or 200 + error:InternalServerError	The server encountered an error while trying to process the request. You could retry the same request following the requirements listed in "Timeout" (see row above). If the error persists, please contact Firebase support.
Device Message Rate Exceeded 200 + error:DeviceMessageRate Exceeded	The rate of messages to a particular device is too high. If an Apple app sends messages at a rate exceeding APNs limits, it may receive this error message

Reduce the number of messages sent to this device and use exponential backoff to retry sending.

Topics Message Rate Exceeded	200 + error:TopicsMessageRate Exceeded The rate of messages to subscribers to a particular topic is too high. Reduce the number of messages sent for this topic and use exponential backoff to retry sending.
*/
