package crocospans

import (
	"bytes"
	"math/rand"
	"net/http"
	"strconv"
	sync "sync"
	"unsafe"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"go.k6.io/k6/lib/netext/httpext"
	"go.k6.io/k6/metrics"
	"go.k6.io/k6/output"
)

// Output implements the k6 output.Output interface
type Output struct {
	config Config

	testRunID int64
	orgID     int64

	httpClient *http.Client

	bufferLock sync.Mutex
	buffer     []*httpext.Trail

	periodicFlusher *output.PeriodicFlusher
	logger          logrus.FieldLogger
}

var _ output.Output = new(Output)

// New creates an instance of the output
func New(p output.Params) (*Output, error) {
	conf, err := NewConfig(p)
	if err != nil {
		return nil, err
	}

	return &Output{
		config:     conf,
		logger:     p.Logger.WithField("component", "xk6-crocospans-output"),
		httpClient: http.DefaultClient, // TODO: some options here?
	}, nil
}

func (o *Output) Description() string {
	return "xk6-crocospans: " + o.config.Endpoint
}

// AddMetricSamples adds the given metric samples to the internal buffer.
func (o *Output) AddMetricSamples(samples []metrics.SampleContainer) {
	if len(samples) == 0 {
		return
	}
	o.bufferLock.Lock()
	defer o.bufferLock.Unlock()
	for _, s := range samples {
		// Only collect HTTP request samples for now
		// TODO: do some sort of sampling or processing?
		if httpSample, ok := s.(*httpext.Trail); ok {
			o.buffer = append(o.buffer, httpSample)
		}
	}
}

func (o *Output) Stop() error {
	o.logger.Debug("Stopping...")
	defer o.logger.Debug("Stopped!")
	o.periodicFlusher.Stop()

	// TODO: do we need to do something here?

	return nil
}

func (o *Output) Start() error {
	o.logger.Debug("Starting...")

	// TODO: initial set up we need to do? get the test run ID and org id somehow?
	o.testRunID = 10000 + rand.Int63n(99999-10000)
	o.orgID = 123
	o.logger.Infof("TestRunID: %d, OrgID: %d", o.testRunID, o.orgID)

	pf, err := output.NewPeriodicFlusher(o.config.PushInterval, o.flushMetrics)
	if err != nil {
		return err
	}
	o.logger.Debug("Started!")
	o.periodicFlusher = pf

	return nil
}

func (o *Output) flushMetrics() {
	o.bufferLock.Lock()
	bufferedTrails := o.buffer
	o.buffer = make([]*httpext.Trail, 0, len(bufferedTrails)) // TODO: optimize like output.SampleBuffer?
	o.bufferLock.Unlock()

	// TODO: do some sort of sampling or processing?

	requests := make([]*Request, 0, len(bufferedTrails))

	for _, trail := range bufferedTrails {
		traceID, hasTrace := trail.Metadata["trace_id"]
		if !hasTrace {
			continue
		}

		totalDuration := trail.Blocked + trail.ConnDuration + trail.Duration
		startTime := trail.EndTime.Add(-totalDuration)

		getTag := func(name string) string {
			val, _ := trail.Tags.Get(name)
			return val
		}

		strStatus := getTag("status")
		status, err := strconv.ParseInt(strStatus, 10, 64)
		if err != nil {
			o.logger.Warnf("unexpected error parsing status '%s': %w", strStatus, err)
			continue
		}

		req := &Request{
			TestRunID:         o.testRunID,
			StartTimeUnixNano: uint64(startTime.UnixNano()),
			EndTimeUnixNano:   uint64(trail.EndTime.UnixNano()),
			Group:             getTag("group"),
			Scenario:          getTag("scenario"),
			TraceID:           traceID,
			HTTPUrl:           getTag("url"),
			HTTPMethod:        getTag("method"),
			HTTPStatus:        status,
		}

		requests = append(requests, req)
	}

	md := &RequestBatch{
		// TODO: FIXME: unsafe.Sizeof() here is almost certainly a bug and both
		// Count and SizeBytes should be unnecessary
		SizeBytes: int64(unsafe.Sizeof(requests)),
		Count:     int64(len(requests)),
		Requests:  requests,
	}

	mm, err := proto.Marshal(md)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal request metadata")
	}

	rq, _ := http.NewRequest("POST", o.config.Endpoint, bytes.NewBuffer(mm))
	rq.Header.Add("X-Scope-OrgID", strconv.Itoa(int(o.orgID)))
	_, err = o.httpClient.Do(rq)
	if err != nil {
		logrus.WithError(err).Error("Failed to send request metadata")
	}

}
