package tracing

import (
	"math/rand"

	"github.com/dop251/goja"
	"github.com/grafana/xk6-distributed-tracing/client"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/js/modules"
	k6HTTP "go.k6.io/k6/js/modules/k6/http"
)

const version = "0.1.1"

func init() {
	modules.Register("k6/x/tracing", New())
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

type Options struct {
	Endpoint   string
	Propagator string
}

var (
	initialized bool = false
	testRunID   int64
)

func (t *DistributedTracing) http(call goja.ConstructorCall) *goja.Object {
	rt := t.vu.Runtime()

	obj := call.Argument(0).ToObject(rt)

	opts := Options{
		Endpoint: obj.Get("endpoint").ToString().String(),
	}

	if !initialized {
		initialized = true
		testRunID = int64(100000000000 + rand.Intn(999999999999-100000000000))
		logrus.Info("Crocospans testRunId: ", testRunID)
	}

	tracingClient := client.New(t.vu, t.httpRequest, opts.Endpoint, testRunID)

	return rt.ToValue(tracingClient).ToObject(rt)
}
