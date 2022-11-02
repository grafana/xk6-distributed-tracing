package tracing

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/grafana/xk6-distributed-tracing/client"
	crocospans "github.com/grafana/xk6-distributed-tracing/cloud"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	k6HTTP "go.k6.io/k6/js/modules/k6/http"
	"go.k6.io/k6/output"
)

const version = "0.2.0"

func init() {
	modules.Register("k6/x/tracing", New())

	output.RegisterExtension("xk6-crocospans", func(p output.Params) (output.Output, error) {
		return crocospans.New(p)
	})
}

type (
	// RootModule is the global module instance that will create DistributedTracing
	// instances for each VU.
	RootModule struct{}

	DistributedTracing struct {
		// modules.VU provides some useful methods for accessing internal k6
		// objects like the global context, VU state and goja runtime.
		vu          modules.VU
		httpRequest client.HttpRequestFunc
	}
)

// Ensure the interfaces are implemented correctly.
var (
	_ modules.Instance = &DistributedTracing{}
	_ modules.Module   = &RootModule{}
)

// New returns a pointer to a new RootModule instance.
func New() *RootModule {
	return &RootModule{}
}

// NewModuleInstance implements the modules.Module interface and returns
// a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	r := k6HTTP.New().NewModuleInstance(vu).Exports().Default.(*goja.Object).Get("request")
	var requestFunc client.HttpRequestFunc
	err := vu.Runtime().ExportTo(r, &requestFunc)
	if err != nil {
		panic(err)
	}
	return &DistributedTracing{vu: vu, httpRequest: requestFunc}
}

// Exports implements the modules.Instance interface and returns the exports
// of the JS module.
func (c *DistributedTracing) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"Http":    c.http,
			"version": version,
		},
	}
}

func (t *DistributedTracing) parseClientOptions(val goja.Value) (client.Options, error) {
	rt := t.vu.Runtime()
	opts := client.Options{
		Propagator: client.PropagatorW3C,
	}

	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return opts, nil
	}

	params := val.ToObject(rt)
	for _, k := range params.Keys() {
		switch k {
		case "propagator":
			opts.Propagator = params.Get(k).ToString().String()
			//TODO: validate
		default:
			return opts, fmt.Errorf("unknown HTTP tracing option '%s'", k)
		}
	}
	return opts, nil
}

func (t *DistributedTracing) http(call goja.ConstructorCall) *goja.Object {
	rt := t.vu.Runtime()
	opts, err := t.parseClientOptions(call.Argument(0))
	if err != nil {
		common.Throw(rt, err)
	}

	return rt.ToValue(client.New(t.vu, t.httpRequest, opts)).ToObject(rt)
}
