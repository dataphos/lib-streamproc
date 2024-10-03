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

package config

import (
	"time"

	"github.com/kkyr/fig"
)

type Config struct {
	Consumer   Consumer `toml:"consumer"`
	RunOptions RunOptions
}

// Read returns a Config structure for the given path to a .toml file.
func Read(filename string) (Config, error) {
	cfg := Default()

	if err := fig.Load(&cfg, fig.File(filename), fig.Tag("toml"), fig.UseEnv("")); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate validates CentralConsumer struct.
func Validate(cfg *Config) error {
	return validate(cfg, "Config.")
}

// Default returns a Config object with default values.
func Default() Config {
	return Config{
		Consumer: Consumer{
			Kafka:      DefaultKafka(),
			Pubsub:     DefaultPubsub(),
			ServiceBus: DefaultServiceBus(),
			Jetstream:  DefaultJetstream(),
			Pulsar:     DefaultPulsar(),
		},
		RunOptions: RunOptions{
			ErrThreshold: 50,
			ErrInterval:  1 * time.Minute,
			NumRetries:   0,
		},
	}
}

func DefaultKafka() KafkaConsumerConfig {
	return KafkaConsumerConfig{
		Settings: KafkaConsumerSettings{
			MinBytes:             100,
			MaxWait:              5 * time.Second,
			MaxBytes:             10485760,
			MaxConcurrentFetches: 3,
			MaxPollRecords:       100,
			Transactional:        false,
		},
	}
}

func DefaultPubsub() PubsubConsumerConfig {
	return PubsubConsumerConfig{
		Settings: PubsubConsumerSettings{
			MaxExtension:           30 * time.Minute,
			MaxExtensionPeriod:     3 * time.Minute,
			MaxOutstandingMessages: 1000,
			MaxOutstandingBytes:    419430400,
			NumGoroutines:          10,
		},
	}
}

func DefaultServiceBus() ServiceBusConsumerConfig {
	return ServiceBusConsumerConfig{
		Settings: ServiceBusConsumerSettings{
			BatchSize: 100,
		},
	}
}

func DefaultJetstream() JetstreamConsumerConfig {
	return JetstreamConsumerConfig{
		Settings: JetstreamConsumerSettings{
			BatchSize: 100,
		},
	}
}

func DefaultPulsar() PulsarConsumerConfig {
	return PulsarConsumerConfig{
		Settings: PulsarConsumerSettings{
			ConnectionTimeout:          5 * time.Second,
			OperationTimeout:           30 * time.Second,
			NackRedeliveryDelay:        30 * time.Second,
			MaxConnectionsPerBroker:    1,
			SubscriptionType:           "Exclusive",
			ReceiverQueueSize:          1000,
			MaxReconnectToBroker:       nil,
			TLSTrustCertsFilePath:      "",
			TLSAllowInsecureConnection: true,
		},
	}
}
