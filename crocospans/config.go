package crocospans

import (
	"fmt"
	"strconv"
	"time"

	"go.k6.io/k6/output"
)

// Config is the config for the crocospans output.
type Config struct {
	Endpoint     string
	PushInterval time.Duration
	OrgID        int64
	Token        string

	// TODO: add other config fields?
}

// NewConfig creates a new Config instance from the provided output.Params
func NewConfig(params output.Params) (Config, error) {
	cfg := Config{
		// TODO: add default Endpoint value
		PushInterval: 1 * time.Second,
	}

	if params.ConfigArgument != "" {
		cfg.Endpoint = params.ConfigArgument
	} else if val, ok := params.Environment["XK6_CROCOSPANS_ENDPOINT"]; ok {
		cfg.Endpoint = val
	}
	if cfg.Endpoint == "" {
		return cfg, fmt.Errorf("missing crocospans endpoint, use '--out xk6-crocospans=http://endpoint' or the XK6_CROCOSPANS_ENDPOINT env var")
	}

	if val, ok := params.Environment["XK6_CROCOSPANS_PUSH_INTERVAL"]; ok {
		var err error
		cfg.PushInterval, err = time.ParseDuration(val)
		if err != nil {
			return cfg, fmt.Errorf("error parsing environment variable 'XK6_CROCOSPANS_PUSH_INTERVAL': %w", err)
		}
	}

	if val, ok := params.Environment["XK6_CROCOSPANS_ORG_ID"]; ok {
		var err error
		cfg.OrgID, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			return cfg, fmt.Errorf("error parsing environment variable 'XK6_CROCOSPANS_ORG_ID': %w", err)
		}
	} else {
		return cfg, fmt.Errorf("XK6_CROCOSPANS_ORG_ID is required")
	}

	if val, ok := params.Environment["XK6_CROCOSPANS_TOKEN"]; ok {
		cfg.Token = val
	} else if val, ok := params.Environment["K6_CLOUD_TOKEN"]; ok {
		cfg.Token = val
	} else {
		return cfg, fmt.Errorf("XK6_CROCOSPANS_TOKEN or K6_CLOUD_TOKEN is required")
	}

	// TODO: add more validation and options

	return cfg, nil
}
