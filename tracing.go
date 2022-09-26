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
	Crocospans string
}

var initialized bool = false
var testRunID int

func (*JsModule) XHttp(ctx *context.Context, opts Options) interface{} {
	if !initialized {
		initialized = true
		testRunID = 100000000000 + rand.Intn(999999999999-100000000000)
		logrus.Info("CrocoSpans test run id: ", testRunID)
	}
	rt := common.GetRuntime(*ctx)
	tracingClient := New(opts.Crocospans, testRunID)
	return common.Bind(rt, tracingClient, ctx)
}
