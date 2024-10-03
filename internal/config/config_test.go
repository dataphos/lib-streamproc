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

package config_test

import (
	"testing"
	"time"

	"go.uber.org/multierr"
	"gotest.tools/assert"

	"github.com/dataphos/lib-streamproc/internal/config"
	"github.com/dataphos/lib-streamproc/internal/errtemplates"
)

var (
	mockServiceBus        = "./testdata/servicebus_test.toml"
	mockServiceBusInvalid = "./testdata/servicebus_invalid_test.toml"
	mockJetStream         = "./testdata/jetstream_test.toml"
	mockJetStreamInvalid  = "./testdata/jetstream_invalid_test.toml"
	mockPubSub            = "./testdata/pubsub_test.toml"
	mockPubSubDefault     = "./testdata/pubsubdefault_test.toml"
	mockPubSubInvalid     = "./testdata/pubsub_invalid_test.toml"
	mockPulsar            = "./testdata/pulsar_test.toml"
	mockPulsarDefault     = "./testdata/pulsardefault_test.toml"
	mockPulsarInvalid     = "./testdata/pulsar_invalid_test.toml"
	mockKafka             = "./testdata/kafka_test.toml"
	mockKafkaDefault      = "./testdata/kafkadefault_test.toml"
	mockKafkaInvalid      = "./testdata/kafka_invalid_test.toml"
	mockFile              = "./testdata/mockFile"
	mockKafkaPort         = "localhost:9092"

	expectedServiceBusConsumerConfig = config.ServiceBusConsumerConfig{
		"sb:/mock_url",
		"test",
		"test",
		expectedServiceBusConsumerSettings,
	}
	expectedServiceBusConsumerSettings = config.ServiceBusConsumerSettings{BatchSize: 10}

	expectedPubsubConsumerConfig = config.PubsubConsumerConfig{
		ProjectID:      "test",
		SubscriptionID: "test",
		Settings:       expectedPubsubConsumerSettings,
	}
	expectedPubsubConsumerSettings = config.PubsubConsumerSettings{
		MaxExtension:           10 * time.Minute,
		MaxExtensionPeriod:     2 * time.Minute,
		MaxOutstandingMessages: 1500,
		MaxOutstandingBytes:    2000,
		NumGoroutines:          5,
	}

	expectedPubsubDefaultConsumerConfig = config.PubsubConsumerConfig{
		ProjectID:      "test",
		SubscriptionID: "test",
		Settings:       expectedPubsubDefaultConsumerSettings,
	}
	expectedPubsubDefaultConsumerSettings = config.PubsubConsumerSettings{
		MaxExtension:           30 * time.Minute,
		MaxExtensionPeriod:     3 * time.Minute,
		MaxOutstandingMessages: 1000,
		MaxOutstandingBytes:    419430400,
		NumGoroutines:          10,
	}

	expectedJetstreamConsumerConfig = config.JetstreamConsumerConfig{
		URL:          "nats:/mock_url",
		Subject:      "test",
		ConsumerName: "test",
		Settings:     expectedJetstreamConsumerSettings,
	}
	expectedJetstreamConsumerSettings = config.JetstreamConsumerSettings{
		BatchSize: 20,
	}

	expectedPulsarConsumerConfig = config.PulsarConsumerConfig{
		ServiceURL:   "ps:/mock_url",
		Topic:        "test",
		Subscription: "test",
		TLSConfig:    expectedPulsarTLSConfig,
		Settings:     expectedPulsarConsumerSettings,
	}
	expectedPulsarTLSConfig = config.TLSConfig{
		Enabled:        true,
		ClientCertFile: mockFile,
		ClientKeyFile:  mockFile,
		CaCertFile:     mockFile,
	}
	maxTenReconnectToBroker        = uint(10)
	expectedPulsarConsumerSettings = config.PulsarConsumerSettings{
		ConnectionTimeout:          3 * time.Minute,
		OperationTimeout:           10 * time.Second,
		NackRedeliveryDelay:        30 * time.Second,
		MaxConnectionsPerBroker:    2,
		SubscriptionType:           "Shared",
		ReceiverQueueSize:          500,
		MaxReconnectToBroker:       &maxTenReconnectToBroker,
		TLSTrustCertsFilePath:      "",
		TLSAllowInsecureConnection: true,
	}

	expectedPulsarDefaultConsumer = config.PulsarConsumerConfig{
		ServiceURL:   "ps:/mock_url",
		Topic:        "test",
		Subscription: "test",
		TLSConfig:    config.TLSConfig{},
		Settings:     expectedPulsarDefaultConsumerSettings,
	}

	expectedPulsarDefaultConsumerSettings = config.PulsarConsumerSettings{
		ConnectionTimeout:          5 * time.Second,
		OperationTimeout:           30 * time.Second,
		NackRedeliveryDelay:        30 * time.Second,
		MaxConnectionsPerBroker:    1,
		SubscriptionType:           "Exclusive",
		ReceiverQueueSize:          1000,
		TLSTrustCertsFilePath:      "",
		TLSAllowInsecureConnection: true,
	}

	expectedKafkaConsumer = config.KafkaConsumerConfig{
		Address:    mockKafkaPort,
		TLSConfig:  config.TLSConfig{},
		Topic:      "test",
		GroupID:    "test",
		Kerberos:   expectedKafkaKerberos,
		PlainSASL:  expectedKafkaPlainSSL,
		Prometheus: expectedKafkaPrometheus,
		Settings:   expectedKafkaConsumerSettings,
	}
	expectedKafkaConsumerSettings = config.KafkaConsumerSettings{
		MinBytes:             50,
		MaxWait:              10 * time.Second,
		MaxBytes:             1000,
		MaxConcurrentFetches: 5,
		MaxPollRecords:       20,
		Transactional:        true,
	}
	expectedKafkaKerberos = config.KerberosConfig{
		Enabled:    true,
		KeytabPath: mockFile,
		ConfigPath: mockFile,
		Realm:      mockFile,
		Service:    mockFile,
		Username:   mockFile,
	}
	expectedKafkaPlainSSL = config.PlainSASLConfig{
		Enabled: true,
		Zid:     "test",
		User:    "test",
		Pass:    "test",
	}
	expectedKafkaPrometheus = config.PrometheusConfig{
		Enabled:    true,
		Namespace:  "test",
		Registerer: nil,
		Gatherer:   nil,
	}

	expectedKafkaDefaultConsumer = config.KafkaConsumerConfig{
		Address:    mockKafkaPort,
		TLSConfig:  config.TLSConfig{},
		Topic:      "test",
		GroupID:    "test",
		Kerberos:   config.KerberosConfig{},
		PlainSASL:  config.PlainSASLConfig{},
		Prometheus: config.PrometheusConfig{},
		Settings:   expectedKafkaDefaultConsumerSettings,
	}
	expectedKafkaDefaultConsumerSettings = config.KafkaConsumerSettings{
		MinBytes:             100,
		MaxWait:              5 * time.Second,
		MaxBytes:             10485760,
		MaxConcurrentFetches: 3,
		MaxPollRecords:       100,
		Transactional:        false,
	}

	nilErrorExpectedTemplate = "expected error is nil, got %q instead"
)

func testInvalidConfiguration(t *testing.T, confPath string, expectedErrors ...error) {
	t.Helper()

	cfg, err := config.Read(confPath)
	assert.NilError(t, err, nilErrorExpectedTemplate, err)

	err = config.Validate(&cfg)

	for _, errExp := range expectedErrors {
		assert.ErrorContains(t, err, errExp.Error())
	}

	multiErr := multierr.Errors(err)
	assert.Check(t, len(multiErr) == len(expectedErrors),
		"expected %d validation errors, got %d", len(multiErr), len(expectedErrors))
}

func testValidConfiguration(t *testing.T, configPath string) config.Config {
	t.Helper()

	cfg, err := config.Read(configPath)
	assert.NilError(t, err, nilErrorExpectedTemplate, err)

	err = config.Validate(&cfg)
	assert.NilError(t, err, nilErrorExpectedTemplate, err)

	return cfg
}

func TestConfigureServiceBus(t *testing.T) {
	cfg := testValidConfiguration(t, mockServiceBus)
	assert.DeepEqual(t, expectedServiceBusConsumerConfig, cfg.Consumer.ServiceBus)
}

func TestConfigureServiceBusInvalid(t *testing.T) {
	// Validation should fail because the connection string is blank and batch size is negative.
	testInvalidConfiguration(t, mockServiceBusInvalid,
		errtemplates.URLTagFail("consumer.servicebus.connection_string", ""),
		errtemplates.MinTagFail("consumer.servicebus.batch_size", "0", "1"))
}

func TestConfigurePubSub(t *testing.T) {
	cfg := testValidConfiguration(t, mockPubSub)
	assert.DeepEqual(t, expectedPubsubConsumerConfig, cfg.Consumer.Pubsub)
}

func TestConfigurePubSubDefault(t *testing.T) {
	cfg := testValidConfiguration(t, mockPubSubDefault)
	assert.DeepEqual(t, expectedPubsubDefaultConsumerConfig, cfg.Consumer.Pubsub)
}

func TestConfigurePubSubInvalid(t *testing.T) {
	// An error is expected because the number of goroutines is set to 0.
	testInvalidConfiguration(t, mockPubSubInvalid,
		errtemplates.MinTagFail("consumer.pubsub.num_goroutines", "0", "1"))
}

func TestConfigureJetstream(t *testing.T) {
	cfg := testValidConfiguration(t, mockJetStream)
	assert.DeepEqual(t, expectedJetstreamConsumerConfig, cfg.Consumer.Jetstream)
}

func TestConfigureJetstreamInvalid(t *testing.T) {
	// Validation should fail as no url is specified and batchSize is negative.
	testInvalidConfiguration(t, mockJetStreamInvalid,
		errtemplates.MinTagFail("consumer.jetstream.batch_size", "-5", "1"),
		errtemplates.URLTagFail("consumer.jetstream.url", ""))
}

func TestConfigurePulsar(t *testing.T) {
	cfg := testValidConfiguration(t, mockPulsar)
	assert.DeepEqual(t, expectedPulsarConsumerConfig, cfg.Consumer.Pulsar)
}

func TestConfigurePulsarDefault(t *testing.T) {
	cfg := testValidConfiguration(t, mockPulsarDefault)
	assert.DeepEqual(t, expectedPulsarDefaultConsumer, cfg.Consumer.Pulsar)
}

func TestConfigurePulsarInvalid(t *testing.T) {
	// Validation should fail as the SubscriptionType doesn't exist and receiver_queue_size is negative.
	testInvalidConfiguration(t, mockPulsarInvalid,
		errtemplates.OneofTagFail("consumer.pulsar.subscription_type", "NonExistant"),
		errtemplates.MinTagFail("consumer.pulsar.receiver_queue_size", "-2", "0"))
}

func TestConfigureKafka(t *testing.T) {
	cfg := testValidConfiguration(t, mockKafka)
	assert.DeepEqual(t, expectedKafkaConsumer, cfg.Consumer.Kafka)
}

func TestConfigureKafkaDefault(t *testing.T) {
	cfg := testValidConfiguration(t, mockKafkaDefault)
	assert.DeepEqual(t, expectedKafkaDefaultConsumer, cfg.Consumer.Kafka)
}

func TestConfigureKafkaInvalid(t *testing.T) {
	// Validation should fail as the address is incorrectly formatted and user for sasl is missing.
	testInvalidConfiguration(t, mockKafkaInvalid,
		errtemplates.RequiredTagFail("consumer.kafka.plain_sasl.user"),
		errtemplates.HostnamePortTagFail("consumer.kafka.address", "kafka:/mock_url"))
}
