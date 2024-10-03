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

// Package config contains the configuration structs and validators used in the project.
package config

import (
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/multierr"

	"github.com/dataphos/lib-streamproc/internal/errtemplates"
)

type RunOptions struct {
	ErrThreshold int64         `toml:"err_threshold"`
	ErrInterval  time.Duration `toml:"err_interval"`
	NumRetries   int           `toml:"num_retries"`
}

type TLSConfig struct {
	Enabled        bool   `toml:"enabled"`
	ClientCertFile string `toml:"client_cert_file" val:"required_if=Enabled true,omitempty,file"`
	ClientKeyFile  string `toml:"client_key_file" val:"required_if=Enabled true,omitempty,file"`
	CaCertFile     string `toml:"ca_cert_file" val:"required_if=Enabled true,omitempty,file"`
}

type Consumer struct {
	Type       string                   `toml:"type" val:"oneof=kafka pubsub servicebus jetstream pulsar"`
	Kafka      KafkaConsumerConfig      `toml:"kafka"`
	Pubsub     PubsubConsumerConfig     `toml:"pubsub"`
	ServiceBus ServiceBusConsumerConfig `toml:"servicebus"`
	Jetstream  JetstreamConsumerConfig  `toml:"jetstream"`
	Pulsar     PulsarConsumerConfig     `toml:"pulsar"`
}

type KafkaConsumerConfig struct {
	Address    string                `toml:"address"`
	TLSConfig  TLSConfig             `toml:"tls_config"`
	Topic      string                `toml:"topic"`
	GroupID    string                `toml:"group_id"`
	Kerberos   KerberosConfig        `toml:"kerberos"`
	PlainSASL  PlainSASLConfig       `toml:"plain_sasl"`
	Prometheus PrometheusConfig      `toml:"prometheus"`
	Settings   KafkaConsumerSettings `toml:"settings"`
}

type KafkaConsumerSettings struct {
	MinBytes             int           `toml:"min_bytes"`
	MaxWait              time.Duration `toml:"max_wait"`
	MaxBytes             int           `toml:"max_bytes"`
	MaxConcurrentFetches int           `toml:"max_concurrent_fetches"`
	MaxPollRecords       int           `toml:"max_poll_records"`
	Transactional        bool          `toml:"transactional"`
}

type KerberosConfig struct {
	Enabled    bool   `toml:"enabled"`
	KeytabPath string `toml:"key_tab_path"`
	ConfigPath string `toml:"config_path"`
	Realm      string `toml:"realm"`
	Service    string `toml:"service"`
	Username   string `toml:"username"`
}

type PlainSASLConfig struct {
	Enabled bool   `toml:"enabled"`
	Zid     string `toml:"zid"`
	User    string `toml:"user"`
	Pass    string `toml:"pass"`
}

type PrometheusConfig struct {
	Enabled    bool   `toml:"enabled"`
	Namespace  string `toml:"namespace"`
	Registerer prometheus.Registerer
	Gatherer   prometheus.Gatherer
}

type PubsubConsumerConfig struct {
	ProjectID      string                 `toml:"project_id"`
	SubscriptionID string                 `toml:"subscription_id"`
	Settings       PubsubConsumerSettings `toml:"settings"`
}

type PubsubConsumerSettings struct {
	MaxExtension           time.Duration `toml:"max_extension"`
	MaxExtensionPeriod     time.Duration `toml:"max_extension_period"`
	MaxOutstandingMessages int           `toml:"max_outstanding_messages"`
	MaxOutstandingBytes    int           `toml:"max_outstanding_bytes"`
	NumGoroutines          int           `toml:"num_goroutines"`
}

type ServiceBusConsumerConfig struct {
	ConnectionString string                     `toml:"connection_string"`
	Topic            string                     `toml:"topic"`
	Subscription     string                     `toml:"subscription"`
	Settings         ServiceBusConsumerSettings `toml:"settings"`
}

type ServiceBusConsumerSettings struct {
	BatchSize int `toml:"batch_size"`
}

type JetstreamConsumerConfig struct {
	URL          string                    `toml:"url"`
	Subject      string                    `toml:"subject"`
	ConsumerName string                    `toml:"consumer_name"`
	Settings     JetstreamConsumerSettings `toml:"settings"`
}

type JetstreamConsumerSettings struct {
	BatchSize int `toml:"batch_size"`
}

type PulsarConsumerConfig struct {
	ServiceURL   string                 `toml:"service_url"`
	Topic        string                 `toml:"topic"`
	Subscription string                 `toml:"subscription"`
	TLSConfig    TLSConfig              `toml:"tls_config"`
	Settings     PulsarConsumerSettings `toml:"settings"`
}

type PulsarConsumerSettings struct {
	ConnectionTimeout          time.Duration `toml:"connection_timeout"`
	OperationTimeout           time.Duration `toml:"operation_timeout"`
	NackRedeliveryDelay        time.Duration `toml:"nack_redelivery_delay"`
	MaxConnectionsPerBroker    int           `toml:"max_connections_per_broker"`
	SubscriptionType           string        `toml:"subscription_type"`
	ReceiverQueueSize          int           `toml:"receiver_queue_size"`
	MaxReconnectToBroker       *uint         `toml:"max_reconnect_to_broker"`
	TLSTrustCertsFilePath      string        `toml:"tls_trust_certs_file_path"`
	TLSAllowInsecureConnection bool          `toml:"tls_allow_insecure_connection" `
}

func validate(cfg interface{}, prefix string) error {
	validate := validator.New()
	validate.SetTagName("val")

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("toml"), ",", 2)[0]
		if name == "-" {
			return ""
		}

		return name
	})

	validate.RegisterStructValidation(
		ConsumerStructLevelValidator,
		KafkaConsumerConfig{},
		PubsubConsumerConfig{},
		ServiceBusConsumerConfig{},
		JetstreamConsumerConfig{},
		PulsarConsumerConfig{},
	)

	if err := validate.Struct(cfg); err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			return err
		}

		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			var errCombined error

			for _, vErr := range validationErrors {
				fieldName := strings.TrimPrefix(vErr.Namespace(), prefix)

				switch vErr.Tag() {
				case "required", "required_if":
					errCombined = multierr.Append(errCombined, errtemplates.RequiredTagFail(fieldName))
				case "file":
					errCombined = multierr.Append(errCombined, errtemplates.FileTagFail(fieldName, vErr.Value()))
				case "url":
					errCombined = multierr.Append(errCombined, errtemplates.URLTagFail(fieldName, vErr.Value()))
				case "oneof":
					errCombined = multierr.Append(errCombined, errtemplates.OneofTagFail(fieldName, vErr.Value()))
				case "hostname_port":
					errCombined = multierr.Append(errCombined, errtemplates.HostnamePortTagFail(fieldName, vErr.Value()))
				case "min":
					errCombined = multierr.Append(errCombined, errtemplates.MinTagFail(fieldName, vErr.Value(), vErr.Param()))
				case "max":
					errCombined = multierr.Append(errCombined, errtemplates.MaxTagFail(fieldName, vErr.Value(), vErr.Param()))
				case "eq":
					errCombined = multierr.Append(errCombined, errtemplates.EqTagFail(fieldName, vErr.Value()))
				case "lte":
					errCombined = multierr.Append(errCombined, errtemplates.LteTagFail(fieldName, vErr.Value(), vErr.Param()))
				case "gte":
					errCombined = multierr.Append(errCombined, errtemplates.GteTagFail(fieldName, vErr.Value(), vErr.Param()))
				case "gtcsfield":
					errCombined = multierr.Append(errCombined, errtemplates.GtFieldTagFail(fieldName, vErr.Value()))
				default:
					errCombined = multierr.Append(errCombined, vErr)
				}
			}

			return errCombined
		}
	}

	return nil
}

// ConsumerStructLevelValidator is a custom validator which validates broker structure.
func ConsumerStructLevelValidator(structLevel validator.StructLevel) {
	source, _ := structLevel.Parent().Interface().(Consumer)
	validate := validator.New()

	switch consumer := structLevel.Current().Interface().(type) {
	case KafkaConsumerConfig:
		if source.Type == "kafka" {
			if err := validate.Var(consumer.Address, "hostname_port"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("address", "", valErr)
				}
			}

			if err := validate.Var(consumer.Topic, "required"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("topic", "", valErr)
				}
			}

			if err := validate.Var(consumer.GroupID, "required"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("group_id", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.MinBytes, "min=10"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("min_bytes", "", valErr)
				}
			}

			if err := validate.VarWithValue(consumer.Settings.MaxBytes, consumer.Settings.MinBytes, "gtcsfield"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("max_bytes", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.MaxConcurrentFetches, "min=0"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("max_concurrent_fetches", "", valErr)
				}
			}

			if consumer.PlainSASL.Enabled {
				if err := validate.Var(consumer.PlainSASL.User, "required"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("plain_sasl.user", "", valErr)
					}
				}

				if err := validate.Var(consumer.PlainSASL.Pass, "required"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("plainSASL.pass", "", valErr)
					}
				}
			}

			if consumer.Prometheus.Enabled {
				if err := validate.Var(consumer.Prometheus.Namespace, "required"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("prometheus.namespace", "", valErr)
					}
				}
			}

			if consumer.Kerberos.Enabled {
				if err := validate.Var(consumer.Kerberos.KeytabPath, "file"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("kerberos.key_tab_path", "", valErr)
					}
				}

				if err := validate.Var(consumer.Kerberos.ConfigPath, "file"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("kerberos.config_path", "", valErr)
					}
				}

				if err := validate.Var(consumer.Kerberos.Realm, "required"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("kerberos.realm", "", valErr)
					}
				}

				if err := validate.Var(consumer.Kerberos.Service, "required"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("kerberos.service", "", valErr)
					}
				}

				if err := validate.Var(consumer.Kerberos.Username, "required"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("kerberos.username", "", valErr)
					}
				}
			}
		}

	case PubsubConsumerConfig:
		if source.Type == "pubsub" {
			if err := validate.Var(consumer.ProjectID, "required"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("project_id", "", valErr)
				}
			}

			if err := validate.Var(consumer.SubscriptionID, "required"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("subscription_id", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.MaxExtension, "min=10"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("max_extension", "", valErr)
				}
			}

			if consumer.Settings.MaxExtensionPeriod != 0 {
				if err := validate.Var(consumer.Settings.MaxExtensionPeriod, "gte=10s"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("max_extension_period", "", valErr)
					}
				}

				if err := validate.Var(consumer.Settings.MaxExtensionPeriod, "lte=600s"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("max_extension_period", "", valErr)
					}
				}
			}

			if err := validate.Var(consumer.Settings.NumGoroutines, "min=1"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("num_goroutines", "", valErr)
				}
			}
		}

	case ServiceBusConsumerConfig:
		if source.Type == "servicebus" {
			if err := validate.Var(consumer.ConnectionString, "url"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("connection_string", "", valErr)
				}
			}

			if err := validate.Var(consumer.Topic, "required"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("topic", "", valErr)
				}
			}

			if err := validate.Var(consumer.Subscription, "required"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("subscription", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.BatchSize, "min=1"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("batch_size", "", valErr)
				}
			}
		}

	case JetstreamConsumerConfig:
		if source.Type == "jetstream" {
			if err := validate.Var(consumer.URL, "url"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("url", "", valErr)
				}
			}

			if err := validate.Var(consumer.Subject, "required"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("subject", "", valErr)
				}
			}

			if err := validate.Var(consumer.ConsumerName, "required"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("consumer_name", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.BatchSize, "min=1"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("batch_size", "", valErr)
				}
			}
		}

	case PulsarConsumerConfig:
		if source.Type == "pulsar" {
			if err := validate.Var(consumer.ServiceURL, "url"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("serviceUrl", "", valErr)
				}
			}

			if err := validate.Var(consumer.Topic, "required"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("topic", "", valErr)
				}
			}

			if err := validate.Var(consumer.Subscription, "required"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("subscription", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.ConnectionTimeout, "min=1s"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("connection_timeout", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.OperationTimeout, "min=1s"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("operation_timeout", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.NackRedeliveryDelay, "min=1s"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("nack_redelivery_delay", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.MaxConnectionsPerBroker, "min=1"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("max_connections_per_broker", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.SubscriptionType, "oneof=Exclusive Shared Failover KeyShared"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("subscription_type", "", valErr)
				}
			}

			if err := validate.Var(consumer.Settings.ReceiverQueueSize, "min=0"); err != nil {
				var valErr validator.ValidationErrors
				if errors.As(err, &valErr) {
					structLevel.ReportValidationErrors("receiver_queue_size", "", valErr)
				}
			}

			if consumer.Settings.TLSTrustCertsFilePath != "" {
				if err := validate.Var(consumer.Settings.TLSTrustCertsFilePath, "file"); err != nil {
					var valErr validator.ValidationErrors
					if errors.As(err, &valErr) {
						structLevel.ReportValidationErrors("tls_trust_certs_file_path", "", valErr)
					}
				}
			}
		}
	}
}
