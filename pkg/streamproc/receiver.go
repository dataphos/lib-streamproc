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
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/dataphos/lib-brokers/pkg/broker"
	"github.com/dataphos/lib-brokers/pkg/brokerutil"
	"github.com/dataphos/lib-retry/pkg/retry"
)

// ReceiverExecutor wraps the given MessageHandler so that the concurrent execution of MessageHandler.HandleMessage
// can be abstracted away, as it handles tracking of errors and preemptive cancellation/notification based on the given FlowControlCallbacks.
type ReceiverExecutor struct {
	MessageHandler MessageHandler
}

// NewReceiverExecutor returns a new instance of ReceiverExecutor.
func NewReceiverExecutor(messageHandler MessageHandler) *ReceiverExecutor {
	return &ReceiverExecutor{
		MessageHandler: messageHandler,
	}
}

// Run runs the underlying MessageHandler, keeping track of errors as they come.
//
// Returns an error if any errors have occurred as part of the processing.
func (e *ReceiverExecutor) Run(ctx context.Context, receiver broker.Receiver, opts ...RunOption) error {
	settings := applyRunOpts(opts...)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var errCount, errSnapshot int64

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	done := make(chan struct{})
	defer func() {
		close(done)
		waitGroup.Wait()
	}()

	go func() {
		defer waitGroup.Done()

		ticker := time.NewTicker(settings.ErrInterval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				atomic.StoreInt64(&errSnapshot, atomic.LoadInt64(&errCount))
			}
		}
	}()

	if err := retry.Do(ctx, retry.WithMaxRetries(settings.NumRetries, retry.Exponential(2*time.Second)), func(ctx context.Context) error {
		return receiver.Receive(ctx, func(ctx context.Context, message broker.Message) {
			if err := e.MessageHandler.HandleMessage(
				ctx,
				Message{
					ID:            message.ID,
					Key:           message.Key,
					Data:          message.Data,
					Attributes:    message.Attributes,
					PublishTime:   message.PublishTime,
					IngestionTime: message.IngestionTime,
				},
			); err != nil {
				message.Nack()
				diff := atomic.AddInt64(&errCount, 1) - atomic.LoadInt64(&errSnapshot)
				if settings.ShouldStopOnProcess(err, diff, settings.ErrThreshold) {
					cancel()
				}

				return
			}
			message.Ack()
		})
	}); err != nil {
		errCount++
		// The Run would be stopped anyway, but we still want to trigger the relevant callback,
		// so that client code can get notified of the event.
		if settings.ShouldStopOnPull(err, errCount-errSnapshot, settings.ErrThreshold) {
			return err
		}
	}

	if errCount > 0 {
		return errors.Errorf("completed with %d errors", errCount)
	}

	return nil
}

// BatchedReceiverExecutor wraps the given BatchHandler so that the concurrent execution of BatchHandler.HandleBatch
// can be abstracted away, as it handles tracking of errors and preemptive cancellation/notification based on the given FlowControlCallbacks.
type BatchedReceiverExecutor struct {
	BatchHandler BatchHandler
}

// NewBatchedReceiverExecutor returns a new instance of ReceiverExecutor.
func NewBatchedReceiverExecutor(batchHandler BatchHandler) *BatchedReceiverExecutor {
	return &BatchedReceiverExecutor{
		BatchHandler: batchHandler,
	}
}

// Run runs the underlying BatchHandler, keeping track of errors as they come.
//
// Returns an error if any errors have occurred as part of the processing.
func (e *BatchedReceiverExecutor) Run(ctx context.Context, receiver broker.BatchedReceiver, opts ...RunOption) error {
	settings := applyRunOpts(opts...)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var errCount, errSnapshot int64

	var waitGroup sync.WaitGroup

	waitGroup.Add(1)

	done := make(chan struct{})
	defer func() {
		close(done)
		waitGroup.Wait()
	}()

	go func() {
		defer waitGroup.Done()

		ticker := time.NewTicker(settings.ErrInterval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				atomic.StoreInt64(&errSnapshot, atomic.LoadInt64(&errCount))
			}
		}
	}()

	if err := retry.Do(ctx, retry.WithMaxRetries(settings.NumRetries, retry.Exponential(2*time.Second)), func(ctx context.Context) error {
		return receiver.ReceiveBatch(ctx, func(ctx context.Context, batch []broker.Message) {
			messages := messageBatchIntoMessages(batch)
			if err := e.BatchHandler.HandleBatch(ctx, messages); err != nil {
				// BatchHandler can optionally return PartiallyProcessedBatchError, so we check if some messages got processed successfully.
				// If such messages exist, we want to Ack them, otherwise the downstream system might experience duplicates,
				// in case processing is not idempotent.
				messagesToNack := batch
				var messagesToAck []broker.Message

				var errPartial *PartiallyProcessedBatchError
				if errors.As(err, &errPartial) {
					messagesToNack = pickPresentBrokerMessages(batch, errPartial.Failed)
					if len(messagesToNack) != len(messages) {
						messagesToAck = pickMissingBrokerMessages(batch, errPartial.Failed)
					}
				}

				brokerutil.NackAll(messagesToNack...)
				if messagesToAck != nil {
					brokerutil.AckAll(messagesToAck...)
				}

				diff := atomic.AddInt64(&errCount, 1) - atomic.LoadInt64(&errSnapshot)
				if settings.ShouldStopOnProcess(err, diff, settings.ErrThreshold) {
					cancel()
				}

				return
			}
			brokerutil.AckAll(batch...)
		})
	}); err != nil {
		errCount++
		if settings.ShouldStopOnPull(err, errCount-errSnapshot, settings.ErrThreshold) {
			return err
		}
	}

	if errCount > 0 {
		return errors.Errorf("completed with %d errors", errCount)
	}

	return nil
}

// messageBatchIntoMessages maps broker.Message instances into Message instances.
func messageBatchIntoMessages(batch []broker.Message) []Message {
	messages := make([]Message, len(batch))
	for i, message := range batch {
		messages[i] = Message{
			ID:            message.ID,
			Key:           message.Key,
			Data:          message.Data,
			Attributes:    message.Attributes,
			PublishTime:   message.PublishTime,
			IngestionTime: message.IngestionTime,
		}
	}

	return messages
}

// pickPresentBrokerMessages returns subset of messages whose index in the slice appears in indices.
func pickPresentBrokerMessages(messages []broker.Message, indices []int) []broker.Message {
	picked := make([]broker.Message, len(indices))
	for i, index := range indices {
		picked[i] = messages[index]
	}

	return picked
}

// pickMissingBrokerMessages returns subset of messages whose index doesn't appear in indices.
func pickMissingBrokerMessages(messages []broker.Message, indices []int) []broker.Message {
	// First populate the set with all possible indices.
	missingIndices := make(map[int]struct{}, len(messages))
	for i := range messages {
		missingIndices[i] = struct{}{}
	}
	// Remove the ones which do appear in indices.
	for _, index := range indices {
		delete(missingIndices, index)
	}

	picked := make([]broker.Message, 0, len(missingIndices))
	for index := range missingIndices {
		picked = append(picked, messages[index])
	}

	return picked
}
