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

// Package streamproc exposes executors, interfaces, data structures, and utility functions which together
// allow users to define a universal processor: a component which can process message streams invariant to the
// specific messaging system in place while still being as performant as possible, taking advantage of
// different approaches messaging systems take.
//
// The source systems are exposed through integration with the [broker] package, the "parent" package of streamproc.
package streamproc

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// Message defines the object retrieved from a messaging system which supports individual message processing.
type Message struct {
	// ID identifies the message (unique across the source topic).
	ID string

	// Key identifies the payload part of the message.
	// Unlike ID, it doesn't have to be unique across the source topic and is mostly used
	// for ordering guarantees.
	Key string

	// Data holds the payload of the message.
	Data []byte

	// Attributes holds the properties commonly used by brokers to pass metadata.
	Attributes map[string]interface{}

	// PublishTime the time this message was published.
	PublishTime time.Time

	// IngestionTime the time this message was received.
	IngestionTime time.Time
}

// MessageHandler is the common interface used in individual message processing.
type MessageHandler interface {
	// HandleMessage handles the processing of a single Message.
	// The method is assumed to be safe to call concurrently.
	HandleMessage(context.Context, Message) error
}

// MessageHandlerFunc convenience type which is the functional equivalent of MessageHandler.
type MessageHandlerFunc func(context.Context, Message) error

// HandleMessage implements MessageHandler by forwarding the call to the underlying MessageHandlerFunc.
func (f MessageHandlerFunc) HandleMessage(ctx context.Context, message Message) error {
	return f(ctx, message)
}

// BatchHandler is the common interface used in batched stream processing.
type BatchHandler interface {
	// HandleBatch handles the processing of a batch of messages.
	HandleBatch(context.Context, []Message) error
}

// BatchHandlerFunc convenience type which is the functional equivalent of BatchHandler.
type BatchHandlerFunc func(context.Context, []Message) error

// HandleBatch implements BatchHandler by forwarding the call to the underlying BatchHandlerFunc.
func (f BatchHandlerFunc) HandleBatch(ctx context.Context, batch []Message) error {
	return f(ctx, batch)
}

type PartiallyProcessedBatchError struct {
	Failed []int
	Err    error
}

func (e *PartiallyProcessedBatchError) Error() string {
	return e.Err.Error()
}

// Unwrap implements the optional Unwrap error method, which allows for proper usage of errors.Is and errors.As.
func (e *PartiallyProcessedBatchError) Unwrap() error {
	return e.Err
}

// Temporary implements the optional Temporary error method, to ensure we don't hide the temporariness of the underlying
// error (in case code checking if this error is temporary doesn't use errors.As but just converts directly).
func (e *PartiallyProcessedBatchError) Temporary() bool {
	var temporary interface {
		Temporary() bool
	}

	// errors.As stops at the first error down the chain which implements temporary
	// this is important because an unrecoverable error could wrap a recoverable one, so we need the "latest" of the two.
	if errors.As(e.Err, &temporary) {
		return temporary.Temporary()
	}

	return true
}

type BatchedExecutor interface {
	Run(ctx context.Context, handler BatchHandler, opts ...RunOption) error
}

type Executor interface {
	Run(ctx context.Context, handler MessageHandler, opts ...RunOption) error
}
