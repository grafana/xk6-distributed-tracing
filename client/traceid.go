package client

import (
	cr "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
)

const (
	PropagatorW3C    = "w3c"
	HeaderNameW3C    = "traceparent"
	PropagatorB3     = "b3"
	HeaderNameB3     = "b3"
	PropagatorJaeger = "jaeger"
	HeaderNameJaeger = "uber-trace-id"
)

func GenerateHeaderBasedOnPropagator(propagator string, traceID string) (http.Header, error) {
	hex8 := RandHexStringRunes(8)

	switch propagator {
	case PropagatorW3C:
		// Docs: https://www.w3.org/TR/trace-context/#version-format
		return http.Header{
			"traceparent": {fmt.Sprintf("00-%s-%s-01", traceID, hex8)},
		}, nil
	case PropagatorB3:
		// Docs: https://github.com/openzipkin/b3-propagation#single-header
		return http.Header{
			"b3": {fmt.Sprintf("%s-%s-1", traceID, hex8)},
		}, nil
	case PropagatorJaeger:
		// Docs: https://www.jaegertracing.io/docs/1.29/client-libraries/#tracespan-identity
		return http.Header{
			"uber-trace-id": {fmt.Sprintf("%s:%s:0:1", traceID, hex8)},
		}, nil
	default:
		return nil, fmt.Errorf("unknown propagator: %s", propagator)
	}
}

var hexRunes = []rune("123456789abcdef")

func RandHexStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = hexRunes[rand.Intn(len(hexRunes))]
	}
	return string(b)
}

const (
	K6Prefix     = 0756 // Being 075 the ASCII code for 'K' :)
	K6Code_Cloud = 12   // To ingest and process the related spans in k6 Cloud.
	K6Code_Local = 33   // To not ingest and process the related spans, b/c they are part of a non-cloud run.
)

// TraceIDs of 16 bytes (128 bits) are supported by w3c, b3 and jaeger
type TraceID struct {
	Prefix            int16
	Code              int8
	UnixTimestampNano uint64
}

func (t *TraceID) IsValid() bool {
	return t.Prefix == K6Prefix && (t.Code == K6Code_Cloud || t.Code == K6Code_Local)
}

func (t *TraceID) IsValidCloud() bool {
	return t.Prefix == K6Prefix && t.Code == K6Code_Cloud
}

func EncodeTraceID(t TraceID) (string, []byte, error) {
	if !t.IsValid() {
		return "", nil, fmt.Errorf("failed to encode traceID: %v", t)
	}

	buf := make([]byte, 16)

	n := binary.PutVarint(buf, int64(t.Prefix))
	n += binary.PutVarint(buf[n:], int64(t.Code))
	n += binary.PutUvarint(buf[n:], t.UnixTimestampNano)

	randomness := make([]byte, 16-len(buf[:n]))
	err := binary.Read(cr.Reader, binary.BigEndian, randomness)
	if err != nil {
		return "", nil, err
	}

	buf = append(buf[:n], randomness[:]...)
	hx := hex.EncodeToString(buf)
	return hx, buf, nil
}

func DecodeTraceID(buf []byte) *TraceID {
	pre, preLen := binary.Varint(buf)
	code, codeLen := binary.Varint(buf[preLen:])
	ts, _ := binary.Uvarint(buf[preLen+codeLen:])

	return &TraceID{
		Prefix:            int16(pre),
		Code:              int8(code),
		UnixTimestampNano: uint64(ts),
	}
}

func FromBytesToString(buf []byte) string {
	return hex.EncodeToString(buf)
}
