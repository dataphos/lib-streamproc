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

package streamproc_test

import (
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/dataphos/lib-streamproc/pkg/streamproc"
)

func TestDefaultRunSettings(t *testing.T) {
	settings := streamproc.DefaultRunSettings

	if settings.ErrThreshold != 50 {
		t.Fatal("expected 50")
	}

	if settings.ErrInterval != time.Minute*1 {
		t.Fatal("expected 1 minute")
	}

	if settings.NumRetries != 0 {
		t.Fatal("expected 0")
	}

	if settings.OnPullErr(errors.New("oops")) != streamproc.FlowControlContinue {
		t.Fatal("expected continue")
	}

	if settings.OnProcessError(errors.New("oops")) != streamproc.FlowControlContinue {
		t.Fatal("expected continue")
	}

	if settings.OnUnrecoverable(syscall.ECONNREFUSED) != streamproc.FlowControlStop {
		t.Fatal("expected stop")
	}

	if settings.OnThresholdReached(errors.New("oops"), 10, 5) != streamproc.FlowControlStop {
		t.Fatal("expected stop")
	}
}

func TestLoadRunOptionsFromEnv(t *testing.T) {
	tt := []struct { //nolint:varnamelen //style guide convention.
		name       string
		envs       map[string]string
		result     streamproc.RunSettings
		shouldFail bool
	}{
		{
			"all settings set",
			map[string]string{
				streamproc.ErrIntervalEnvKey:  "1m",
				streamproc.ErrThresholdEnvKey: "100",
			},
			streamproc.RunSettings{
				ErrThreshold: 100,
				ErrInterval:  1 * time.Minute,
			},
			false,
		},
		{
			"none set",
			map[string]string{},
			streamproc.RunSettings{},
			false,
		},
		{
			"some set 1",
			map[string]string{
				streamproc.ErrThresholdEnvKey: "99",
			},
			streamproc.RunSettings{
				ErrThreshold: 99,
			},
			false,
		},
		{
			"some set 2",
			map[string]string{
				streamproc.ErrIntervalEnvKey: "1s",
			},
			streamproc.RunSettings{
				ErrInterval: 1 * time.Second,
			},
			false,
		},
		{
			"parsing error 1",
			map[string]string{
				streamproc.ErrIntervalEnvKey:  "a minute",
				streamproc.ErrThresholdEnvKey: "100",
			},
			streamproc.RunSettings{},
			true,
		},
		{
			"parsing error 2",
			map[string]string{
				streamproc.ErrIntervalEnvKey:  "2m",
				streamproc.ErrThresholdEnvKey: "a couple errors",
			},
			streamproc.RunSettings{},
			true,
		},
	}

	for _, tc := range tt {
		tc := tc //nolint:varnamelen //style guide convention.
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				for k := range tc.envs {
					if err := os.Unsetenv(k); err != nil {
						t.Error(err)
					}
				}
			}()

			for k, v := range tc.envs {
				if err := os.Setenv(k, v); err != nil {
					t.Error(err)
				}
			}

			opts, err := streamproc.LoadRunOptionsFromEnv()
			if tc.shouldFail {
				if err == nil {
					t.Error("expected an error")
				}
			} else {
				settings := streamproc.RunSettings{}
				for _, opt := range opts {
					opt(&settings)
				}
				if !reflect.DeepEqual(tc.result, settings) {
					t.Error("expected and actual not the same")
				}
			}
		})
	}
}
