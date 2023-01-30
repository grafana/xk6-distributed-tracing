package client

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type randReaderMock struct{}

func (r randReaderMock) Read(p []byte) (n int, err error) {
	copy(p, []byte{115, 111, 109, 101})

	return 4, nil
}

func TestTraceID_IsValid(t *testing.T) {
	type args struct {
		Prefix int16
		Code   int8
		Time   time.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ReturnsTrueWithValidK6CloudCode",
			args: args{
				Prefix: K6Prefix,
				Code:   K6CloudCode,
				Time:   time.Unix(123456789, 0),
			},
			want: true,
		},
		{
			name: "ReturnsTrueWithValidK6LocalCode",
			args: args{
				Prefix: K6Prefix,
				Code:   K6LocalCode,
				Time:   time.Unix(123456789, 0),
			},
			want: true,
		},
		{
			name: "ReturnsFalseWithInvalidPrefix",
			args: args{
				Prefix: 0,
				Code:   K6CloudCode,
				Time:   time.Unix(123456789, 0),
			},
			want: false,
		},
		{
			name: "ReturnsFalseWithInvalidCode",
			args: args{
				Prefix: K6Prefix,
				Code:   0,
				Time:   time.Unix(123456789, 0),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := TraceID{
				Prefix: tt.args.Prefix,
				Code:   tt.args.Code,
				Time:   tt.args.Time,
			}

			got := tr.IsValid()

			assert.Equalf(t, tt.want, got, "%v.IsValid()", tr)
		})
	}
}

func TestTraceID_IsValidCloud(t *testing.T) {
	type args struct {
		Prefix int16
		Code   int8
		Time   time.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ReturnsTrueWithValidK6CloudCode",
			args: args{
				Prefix: K6Prefix,
				Code:   K6CloudCode,
				Time:   time.Unix(123456789, 0),
			},
			want: true,
		},
		{
			name: "ReturnsFalseWithInvalidCode",
			args: args{
				Prefix: K6Prefix,
				Code:   K6LocalCode,
				Time:   time.Unix(123456789, 0),
			},
			want: false,
		},
		{
			name: "ReturnsFalseWithInvalidPrefix",
			args: args{
				Prefix: 0,
				Code:   K6CloudCode,
				Time:   time.Unix(123456789, 0),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &TraceID{
				Prefix: tt.args.Prefix,
				Code:   tt.args.Code,
				Time:   tt.args.Time,
			}

			got := tr.IsValidCloud()

			assert.Equalf(t, tt.want, got, "%v.IsValidCloud()", tr)
		})
	}
}

func Test_Encode_ReturnsNoErrorAndCorrectlyEncodedStringWithValidTraceID(t *testing.T) {
	traceID := TraceID{
		Prefix: K6Prefix,
		Code:   K6CloudCode,
		Time:   time.Unix(1629191640, 0),
	}

	hx, err := Encode(traceID, &randReaderMock{})

	assert.NoError(t, err)
	assert.Equal(t, "dc071880c0e3d3c5ca869c2d736f6d65", hx)
}

func Test_Encode_ReturnsErrorWithInvalidTraceID(t *testing.T) {
	traceID := TraceID{
		Prefix: 0,
		Code:   1,
	}

	_, err := Encode(traceID, &randReaderMock{})

	assert.Error(t, err)
}
