package ios

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"

	"push-sender/internal/push"

	log "github.com/sirupsen/logrus"
)

// APN service endpoint URLs.
const (
	DevelopmentGateway = "https://api.development.push.apple.com"
	ProductionGateway  = "https://api.push.apple.com"
)

var transport *http.Transport

func init() {
	transport = &http.Transport{
		MaxIdleConnsPerHost: 10,
		MaxIdleConns:        100,
		IdleConnTimeout:     1 * time.Second,
	}

	if err := http2.ConfigureTransport(transport); err != nil {
		log.Errorf("apns cannot configure transport %s", err)
	}
}

type ApnsOptions struct {
	BundleId string
	Cert     string
	Auth     string
}

func (opts *ApnsOptions) Valid() bool {
	return opts.BundleId != "" && opts.Cert != ""
}

type Client struct {
	http     *http.Client
	endpoint string
}

type Response struct {
	Timestamp int64  `json:"timestamp"`
	Reason    string `json:"reason"`
}

// NewClient creates new APNS client based on defined Options.
func NewClient(opts *ApnsOptions) (*Client, error) {
	cert, err := makeCert(opts.Cert, "")

	if err != nil {
		log.Errorf("apns: forriben error %s", err)
		return nil, push.ErrorInvalidKey
	}

	tlsClientConfig := &tls.Config{
		Certificates: []tls.Certificate{*cert},
	}

	if len(cert.Certificate) > 0 {
		tlsClientConfig.BuildNameToCertificate()
	}

	endpoint := ProductionGateway

	if opts.Auth == "dev" {
		endpoint = DevelopmentGateway
	}

	c := &Client{
		http: &http.Client{
			Transport: &http2.Transport{
				TLSClientConfig: tlsClientConfig,
				ReadIdleTimeout: 60 * time.Second,
				DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
					dialer := &net.Dialer{
						Timeout:   60 * time.Second,
						KeepAlive: 1 * time.Second,
					}

					return tls.DialWithDialer(dialer, network, addr, cfg)
				},
			},

			Timeout: 60 * time.Second,
		},
		endpoint: endpoint,
	}

	return c, nil
}

// Send sends Notification to the APN service.
func (c *Client) Send(token string, payload string, opts *ApnsOptions) error {
	req, err := c.prepareRequest(token, payload, opts)
	if err != nil {
		return err
	}
	return c.do(req)
}

func (c *Client) prepareRequest(token string, payload string, opts *ApnsOptions) (*http.Request, error) {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/3/device/%s", c.endpoint, token),
		bytes.NewBuffer([]byte(payload)),
	)

	if err != nil {
		log.Errorf("apns: bad make request %#v", opts)
		return nil, push.ErrorRequest
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apns-topic", opts.BundleId)

	return req, nil
}

func (c *Client) do(req *http.Request) error {
	resp, err := c.http.Do(req)

	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if err != nil {
		log.Errorf("apns: cannot send push to request %s", err)
		return push.ErrorTransportProblem
	}

	if resp.StatusCode == http.StatusOK {
		log.Debugf("apns: succes send %v", resp)
		return nil
	}

	if resp.StatusCode == http.StatusForbidden {
		log.Errorf("apns: forriben error %#v", resp)
		return push.ErrorInvalidKey
	}

	var response Response

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Errorf("anps: strange bad response %v", resp.Body)
		return push.ErrorServiceUnavailable
	}

	log.Errorf("apns: error request %s", response.Reason)

	if response.Reason == "BadDeviceToken" ||
		response.Reason == "MissingDeviceToken" ||
		response.Reason == "Unregistered" {
		return push.ErrorTokenRemoved
	}

	return fmt.Errorf("apns error [%w]", push.PushError(response.Reason))
}

/*
	"BadCollapseID":               ErrBadCollapseID,
	"BadDeviceToken":              ErrBadDeviceToken,
	"BadExpirationDate":           ErrBadExpirationDate,
	"BadMessageId":                ErrBadMessageID,
	"BadPriority":                 ErrBadPriority,
	"BadTopic":                    ErrBadTopic,
	"DeviceTokenNotForTopic":      ErrDeviceTokenNotForTopic,
	"DuplicateHeaders":            ErrDuplicateHeaders,
	"IdleTimeout":                 ErrIdleTimeout,
	"MissingDeviceToken":          ErrMissingDeviceToken,
	"MissingTopic":                ErrMissingTopic,
	"PayloadEmpty":                ErrPayloadEmpty,
	"TopicDisallowed":             ErrTopicDisallowed,
	"BadCertificate":              ErrBadCertificate,
	"BadCertificateEnvironment":   ErrBadCertificateEnvironment,
	"ExpiredProviderToken":        ErrExpiredProviderToken,
	"Forbidden":                   ErrForbidden,
	"InvalidProviderToken":        ErrInvalidProviderToken,
	"MissingProviderToken":        ErrMissingProviderToken,
	"BadPath":                     ErrBadPath,
	"MethodNotAllowed":            ErrMethodNotAllowed,
	"Unregistered":                ErrUnregistered,
	"PayloadTooLarge":             ErrPayloadTooLarge,
	"TooManyProviderTokenUpdates": ErrTooManyProviderTokenUpdates,
	"TooManyRequests":             ErrTooManyRequests,
	"InternalServerError":         ErrInternalServerError,
	"ServiceUnavailable":          ErrServiceUnavailable,
	"Shutdown":                    ErrShutdown,
*/
