# lib-streamproc

lib-streamproc is a Go library that exposes executors, interfaces, data structures, and utility functions which
combined a universal stream processor, invariant to any specific messaging system, while still being as performant as
possible, taking advantage of different approaches messaging systems take.

The source systems are exposed through integration with the [lib-brokers](https://github.com/dataphos/lib-brokers) library,
the "parent" library of lib-streamproc.

## Installation

`go get github.com/dataphos/lib-streamproc`

Note that due to the external dependencies of [lib-brokers](https://github.com/dataphos/lib-brokers), Go version 1.18 or higher is required.

## Important Notice

The latest version of lib-streamproc is unstable, so there might be breaking changes with every release before v1.0.0 is
reached. If you experience bugs or would like to start a discussion about the soundness of the API or would like to
see some additional feature added, contact the Labs team.

## Getting Started

The focus point of lib-streamproc are *executors*, wrappers around [lib-brokers](https://github.com/dataphos/lib-brokers)-compliant consumers,
which abstract away details such as properly (and efficiently) polling the consumer, acknowledging the messages, retry logic and
more sophisticated error handling.

Since the consumer API of the [lib-brokers](https://github.com/dataphos/lib-brokers) library is split into two 
(message-based brokers and log-based brokers), there are multiple
executor implementations, purpose-built for both of these groups.  
For example, `ReceiverExecutor` is specialized for `broker.Receiver` implementations.  
These executors allow for some specific optimizations like [double buffering](https://en.wikipedia.org/wiki/Multiple_buffering),
or *lazy acknowledges*, which minimize (or completely remove) the time the system is blocked waiting for
the broker to respond to acknowledge requests.

### The Message Structure

The core of lib-streamproc is the `Message` structure; although the consumer API is split into two,
the fact lib-streamproc exposes specialized executors which take care of things like acknowledges,
allow them to pass a *unified* message structures onto user-defined functions.

The `Message` struct is defined as follows:

```go
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
```

Users need to only define the `MessageHandler` or `BatchHandler`, depending on if
they perform batch processing or not. The definition of these two interfaces is shown below:

```go
type MessageHandler interface {
  // HandleMessage handles the processing of a single Message.
  // The method is assumed to be safe to call concurrently.
  HandleMessage(context.Context, Message) error
}

type BatchHandler interface {
        // HandleBatch handles the processing of a batch of messages.
        HandleBatch(context.Context, []Message) error
}
```

These two interfaces and the `Message` type is what allows lib-streamproc to
abstract over the [lib-brokers](https://github.com/dataphos/lib-brokers) library and form
a universal processor over messaging systems.

