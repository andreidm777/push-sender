package ios

import (
	"errors"
	"time"

	"push-sender/internal/push"

	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"
)

func init() {
	config.SetDefault("ios_cert_check_timeout", 1200)
}

type BundleClient struct {
	Time   int64
	Cert   string
	Client *Client
}

type IosSender struct {
	Clients map[string]*BundleClient
}

func New() *IosSender {
	return &IosSender{
		Clients: make(map[string]*BundleClient),
	}
}

func (sender *IosSender) Send(token string, payload string, opts *ApnsOptions) error {

	if !opts.Valid() {
		log.Errorf("apns: options not valid %#v", opts)
		return push.ErrorRequest
	}

	if cli, ok := sender.Clients[opts.BundleId]; ok {
		if time.Now().Unix()-cli.Time < int64(config.GetUint64("ios_cert_check_timeout")) || cli.Cert == opts.Cert {
			err := cli.Client.Send(token, payload, opts)
			if !errors.Is(err, push.ErrorTransportProblem) {
				return err
			}
		}
	}

	delete(sender.Clients, opts.BundleId)

	log.Infof("ios: Cert changed [%s]", opts.BundleId)

	b := &BundleClient{
		Time: time.Now().Unix(),
		Cert: opts.Cert,
	}

	newCli, err := NewClient(opts)

	if err != nil {
		log.Errorf("ios: ahtung bad make client %s", err)
		return push.ErrorRequest
	}

	b.Client = newCli

	sender.Clients[opts.BundleId] = b

	return b.Client.Send(token, payload, opts)
}
