// Copyright 2024 Syntio Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package streamproc

import (
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/dataphos/lib-streamproc/internal/errtemplates"
)

// FlowControl determines if the executors should signal the source to stop producing messages.
type FlowControl int64

const (
	// FlowControlContinue used by callbacks to signal to the executor to continue as is.
	FlowControlContinue FlowControl = iota + 1

	// FlowControlStop used by callbacks to signal to the executor to terminate the consuming process.
	FlowControlStop
)

// RunSettings holds all the settings which can be configured on a per-run basis.
type RunSettings struct {
	// ErrThreshold the acceptable amount of unrecoverable message processing errors per ErrInterval.
	// If the threshold is reached, a run is preemptively canceled.
	//
	// A non-positive ErrThreshold is ignored.
	ErrThreshold int64

	// ErrInterval the time interval used to reset the ErrThreshold counter.
	// If no change to the counter is observed in this interval, the counter is reset, as it's safe to
	// assume the system has managed to recover from the erroneous behavior.
	//
	// Only used if ErrThreshold is a positive integer.
	ErrInterval time.Duration

	// NumRetries determines the number of times the executor will repeat erroneous calls to the handler.
	//
	// Keep in mind this may result in duplicates if the messaging system resends messages on acknowledgment timeout.
	//
	// Setting this option will lead record-based executors to stop polling for new messages until the ones which
	// are currently being retry-ed are either successful or the number of retries exceeds NumRetries.
	//
	// NumRetries is set to 0 by default.
	NumRetries int

	// OnPullErr defines a callback function which is called when a pulling error is received.
	OnPullErr func(error) FlowControl

	// OnProcessError defines a callback function which is called on each Pipeliner.Pipeline error.
	OnProcessError func(error) FlowControl

	// OnUnrecoverable defines a callback function which is called on all errors which are inferred to be unrecoverable (most I/O errors, refused connections etc.)
	OnUnrecoverable func(error) FlowControl

	// OnThresholdReached defines a callback function which is called when the threshold defined by RunSettings.ErrThreshold is reached.
	OnThresholdReached func(error, int64, int64) FlowControl
}

// DefaultRunSettings contains the default values of RunSettings.
var DefaultRunSettings = RunSettings{
	ErrThreshold: 50,
	ErrInterval:  1 * time.Minute,
	NumRetries:   0,
	OnPullErr: func(_ error) FlowControl {
		return FlowControlContinue
	},
	OnProcessError: func(_ error) FlowControl {
		return FlowControlContinue
	},
	OnUnrecoverable: func(_ error) FlowControl {
		return FlowControlStop
	},
	OnThresholdReached: func(_ error, _, _ int64) FlowControl {
		return FlowControlStop
	},
}

// RunOption is a convenience type for applying optional settings when running executors.
type RunOption func(*RunSettings)

// WithErrThreshold returns a RunOption which sets RunSettings.ErrThreshold.
func WithErrThreshold(errThreshold int64) RunOption {
	return func(settings *RunSettings) {
		settings.ErrThreshold = errThreshold
	}
}

// WithErrInterval returns a RunOption which sets RunSettings.ErrInterval.
func WithErrInterval(errInterval time.Duration) RunOption {
	return func(settings *RunSettings) {
		settings.ErrInterval = errInterval
	}
}

// WithNumRetires returns a RunOption which sets RunSettings.NumRetries.
func WithNumRetires(numRetries int) RunOption {
	return func(settings *RunSettings) {
		settings.NumRetries = numRetries
	}
}

// OnPullErr returns a RunOption which sets RunSettings.OnPullErr with the given callback.
func OnPullErr(f func(error) FlowControl) RunOption {
	return func(callbacks *RunSettings) {
		callbacks.OnPullErr = f
	}
}

// OnProcessErr returns a RunOption which sets RunSettings.OnProcessError with the given callback.
func OnProcessErr(f func(error) FlowControl) RunOption {
	return func(callbacks *RunSettings) {
		callbacks.OnProcessError = f
	}
}

// OnUnrecoverable returns a RunOption which sets RunSettings.OnUnrecoverable with the given callback.
func OnUnrecoverable(f func(error) FlowControl) RunOption {
	return func(callbacks *RunSettings) {
		callbacks.OnUnrecoverable = f
	}
}

// OnThresholdReached returns a RunOption which sets RunSettings.OnThresholdReached with the given callback.
func OnThresholdReached(f func(error, int64, int64) FlowControl) RunOption {
	return func(callbacks *RunSettings) {
		callbacks.OnThresholdReached = f
	}
}

func applyRunOpts(opts ...RunOption) RunSettings {
	settings := DefaultRunSettings

	for _, opt := range opts {
		opt(&settings)
	}

	return settings
}

func (c RunSettings) ShouldStopOnPull(err error, diff, threshold int64) bool {
	return c.OnPullErr(err) == FlowControlStop || c.shouldStop(err, diff, threshold)
}

func (c RunSettings) ShouldStopOnProcess(err error, diff, threshold int64) bool {
	return c.OnProcessError(err) == FlowControlStop || c.shouldStop(err, diff, threshold)
}

func (c RunSettings) shouldStop(err error, diff, threshold int64) bool {
	return (isUnrecoverable(err) && c.OnUnrecoverable(err) == FlowControlStop) ||
		(threshold > 0 && diff >= threshold && c.OnThresholdReached(err, diff, threshold) == FlowControlStop)
}

// isUnrecoverable checks if the encountered error is considered unrecoverable.
//
// This could happen, for example, if a microservice used in message processing in unresponsive.
func isUnrecoverable(err error) bool {
	// certain error types, like the net.OpError or syscall errors implement this interface.
	var temporary interface {
		Temporary() bool
	}

	// errors.As stops at the first error down the chain which implements temporary
	// this is important because an unrecoverable error could wrap a recoverable one, so we need the "latest" of the two.
	if errors.As(err, &temporary) {
		return !temporary.Temporary()
	}

	return false
}

const (
	ErrThresholdEnvKey = "EXECUTOR_ERR_THRESHOLD"
	ErrIntervalEnvKey  = "EXECUTOR_ERR_INTERVAL"
	NumRetriesEnvKey   = "EXECUTOR_NUM_RETRIES"
)

// LoadRunOptionsFromEnv loads the RunOption instances from the defined environment variables.
func LoadRunOptionsFromEnv() ([]RunOption, error) {
	var opts []RunOption

	if errThresholdStr := os.Getenv(ErrThresholdEnvKey); errThresholdStr != "" {
		errThreshold, err := strconv.ParseInt(errThresholdStr, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, errtemplates.ParsingEnvVariableFailed(ErrThresholdEnvKey))
		}

		opts = append(opts, WithErrThreshold(errThreshold))
	}

	if errIntervalStr := os.Getenv(ErrIntervalEnvKey); errIntervalStr != "" {
		errInterval, err := time.ParseDuration(errIntervalStr)
		if err != nil {
			return nil, errors.Wrap(err, errtemplates.ParsingEnvVariableFailed(ErrIntervalEnvKey))
		}

		opts = append(opts, WithErrInterval(errInterval))
	}

	if numRetriesStr := os.Getenv(NumRetriesEnvKey); numRetriesStr != "" {
		numRetries, err := strconv.Atoi(numRetriesStr)
		if err != nil {
			return nil, errors.Wrap(err, errtemplates.ParsingEnvVariableFailed(NumRetriesEnvKey))
		}

		opts = append(opts, WithNumRetires(numRetries))
	}

	return opts, nil
}
