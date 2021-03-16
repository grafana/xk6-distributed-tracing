package tracing


import (
	"context"
	"github.com/loadimpact/k6/js/common"
	"github.com/loadimpact/k6/js/modules"
	"github.com/simskij/xk6-distributed-tracing/client"
)

const version = "0.0.1"

func init() {
	modules.Register(
		"k6/x/tracing", 
		&TracingModule{
			Version: version,
		})

}

type TracingModule struct {
	Version string
	Http *client.TracingClient
}


func (*TracingModule) XHttp(ctx *context.Context) interface{} {
	rt := common.GetRuntime(*ctx)
	tracingClient := client.New()
	return common.Bind(rt, tracingClient, ctx)
}