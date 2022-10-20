package tracing

import (
	"context"
	"math/rand"

	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

const version = "0.0.2"

func init() {
	modules.Register(
		"k6/x/tracing",
		&JsModule{
			Version: version,
		})
}

// JsModule exposes the tracing client in the javascript runtime
type JsModule struct {
	Version string
	Http    *TracingClient
}

type Options struct {
	Endpoint string
	Org      string
	Token    string
}

var initialized bool = false
var testRunID int

func (*JsModule) XHttp(ctx *context.Context, opts Options) interface{} {
	if !initialized {
		initialized = true
		testRunID = 10000 + rand.Intn(99999-10000)
		logrus.Info("Crocospans testRunID: ", testRunID)
	}
	rt := common.GetRuntime(*ctx)
	tracingClient := New(opts.Endpoint, testRunID, opts.Org, opts.Token)
	return common.Bind(rt, tracingClient, ctx)
}
