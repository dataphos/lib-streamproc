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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/dataphos/lib-brokers/pkg/broker"
	"github.com/dataphos/lib-brokers/pkg/brokerutil"
	"github.com/dataphos/lib-retry/pkg/retry"
)

// RecordExecutor wraps the given Handler so that the iterative execution of MessageHandler.HandleMessage can be abstracted away,
// since it handles tracking of errors and preemptive cancellation/notification based on the given FlowControlCallbacks.
type RecordExecutor struct {
	Handler MessageHandler
}

// NewRecordExecutor returns a new instance of RecordExecutor.
func NewRecordExecutor(handler MessageHandler) *RecordExecutor {
	return &RecordExecutor{
		Handler: handler,
	}
}

// Run runs the underlying Handler, keeping track of errors as they come.
//
// Returns an error if any errors have occurred as part of the processing.
func (e *RecordExecutor) Run(ctx context.Context, iterator broker.RecordIterator, opts ...RunOption) error {
	settings := applyRunOpts(opts...)

	var pendingAckWg sync.WaitGroup
	defer pendingAckWg.Wait()

	stream := brokerutil.StreamifyIterator(ctx, iterator)

	ticker := time.NewTicker(settings.ErrInterval)
	defer ticker.Stop()

	var errCount, errSnapshot int64

RUN:
	for {
		select {
		case <-ticker.C:
			errSnapshot = errCount
		case result, ok := <-stream:
			if !ok {
				break RUN
			}

			record, err := result.Record, result.Err
			if err != nil {
				errCount++
				if settings.ShouldStopOnPull(err, errCount-errSnapshot, settings.ErrThreshold) {
					return err
				}

				continue
			}

			if err = retry.Do(ctx, retry.WithMaxRetries(settings.NumRetries, retry.Exponential(500*time.Millisecond)), func(ctx context.Context) error {
				return e.Handler.HandleMessage(
					ctx,
					Message{
						ID:            partitionAndOffsetIntoID(record.Partition, record.Offset),
						Key:           record.Key,
						Data:          record.Value,
						Attributes:    record.Attributes,
						PublishTime:   record.PublishTime,
						IngestionTime: record.IngestionTime,
					},
				)
			}); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					break RUN
				}

				errCount++
				if settings.ShouldStopOnProcess(err, errCount-errSnapshot, settings.ErrThreshold) {
					return err
				}

				continue
			}
			pendingAckWg.Wait()
			pendingAckWg.Add(1)
			go func() {
				defer pendingAckWg.Done()
				record.Ack()
			}()
		}
	}

	if errCount > 0 {
		return errors.Errorf("completed with %d errors", errCount)
	}

	return nil
}

func partitionAndOffsetIntoID(partition, offset int64) string {
	return strings.Join([]string{strconv.FormatInt(partition, 10), strconv.FormatInt(offset, 10)}, "_")
}

// BatchExecutor wraps the given BatchHandler so that the iterative execution of BatchHandler.HandleBatch can be abstracted away,
// since it handles tracking of errors and preemptive cancellation/notification based on the given FlowControlCallbacks.
type BatchExecutor struct {
	BatchHandler BatchHandler
}

// NewBatchExecutor returns a new instance of BatchExecutor.
func NewBatchExecutor(handler BatchHandler) *BatchExecutor {
	return &BatchExecutor{
		BatchHandler: handler,
	}
}

// Run runs the underlying BatchHandler, keeping track of errors as they come.
//
// Returns an error if any errors have occurred as part of the processing.
func (e *BatchExecutor) Run(ctx context.Context, iterator broker.BatchedIterator, opts ...RunOption) error {
	settings := applyRunOpts(opts...)

	var pendingAckWg sync.WaitGroup
	defer pendingAckWg.Wait()

	stream := brokerutil.StreamifyBatchIterator(ctx, iterator)

	ticker := time.NewTicker(settings.ErrInterval)
	defer ticker.Stop()

	var errCount, errSnapshot int64

RUN:
	for {
		select {
		case <-ticker.C:
			errSnapshot = errCount
		case result, ok := <-stream:
			if !ok {
				break RUN
			}

			batch, err := result.Batch, result.Err
			if err != nil {
				errCount++
				if settings.ShouldStopOnPull(err, errCount-errSnapshot, settings.ErrThreshold) {
					return err
				}

				continue
			}

			messages := recordBatchIntoMessages(batch)
			if err = retry.Do(ctx, retry.WithMaxRetries(settings.NumRetries, retry.Exponential(500*time.Millisecond)), func(ctx context.Context) error {
				if err = e.BatchHandler.HandleBatch(ctx, messages); err != nil {
					var errPartial *PartiallyProcessedBatchError
					if errors.As(err, &errPartial) {
						messages = pickPresentMessages(messages, errPartial.Failed)
					}

					return err
				}

				return nil
			}); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					break RUN
				}

				errCount++
				if settings.ShouldStopOnProcess(err, errCount-errSnapshot, settings.ErrThreshold) {
					return err
				}

				continue
			}
			pendingAckWg.Wait()
			pendingAckWg.Add(1)
			go func() {
				defer pendingAckWg.Done()
				brokerutil.CommitHighestOffsets(batch)
			}()
		}
	}

	if errCount > 0 {
		return errors.Errorf("completed with %d errors", errCount)
	}

	return nil
}

// recordBatchIntoMessages maps all records of the batch slice into Message instances.
func recordBatchIntoMessages(batch []broker.Record) []Message {
	records := make([]Message, len(batch))
	for i, record := range batch {
		records[i] = Message{
			ID:            partitionAndOffsetIntoID(record.Partition, record.Offset),
			Key:           record.Key,
			Data:          record.Value,
			Attributes:    record.Attributes,
			PublishTime:   record.PublishTime,
			IngestionTime: record.IngestionTime,
		}
	}

	return records
}

// pickPresentMessages returns subset of messages whose index in the slice appears in indices.
func pickPresentMessages(messages []Message, indices []int) []Message {
	picked := make([]Message, len(indices))
	for i, index := range indices {
		picked[i] = messages[index]
	}

	return picked
}
