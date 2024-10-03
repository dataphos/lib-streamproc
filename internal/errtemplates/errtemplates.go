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

// Package errtemplates offers convenience functions to standardize error messages and simplify proper error wrapping.
package errtemplates

import (
	"fmt"

	"github.com/pkg/errors"
)

const (
	parsingEnvVariableFailedTemplate = "parsing env variable %s failed"
	requiredTagFailTemplate          = "Validation for '%s' failed: can not be blank"
	fileTagFailTemplate              = "Validation for '%s' failed: '%v' does not exist"
	urlTagFailTemplate               = "Validation for '%s' failed: '%v' incorrect url"
	oneofTagFailTemplate             = "Validation for '%s' failed: '%v' is not one of the options"
	hostnamePortTagFailTemplate      = "Validation for '%s' failed: '%v' incorrect hostname and port"
	minTagFailTemplate               = "Validation for '%s' failed: '%v' value should be greater than '%s'"
	maxTagFailTemplate               = "Validation for '%s' failed: '%v' value should be lower than '%s'"
	eqFailTemplate                   = "Validation for '%s' failed: '%v' incorrect value"
	lteFailTemplate                  = "Validation for '%s' failed: '%v' should be less than '%s'"
	gteFailTemplate                  = "Validation for '%s' failed: '%v' should be greater than '%s'"
	gtFieldTagFailTemplate           = "Validation for '%s' failed: '%v' should be greater than another field's value using gtcs tag validation"
)

// ParsingEnvVariableFailed returns a string stating that the given env variable couldn't be parsed properly.
func ParsingEnvVariableFailed(name string) string {
	return fmt.Sprintf(parsingEnvVariableFailedTemplate, name)
}

// RequiredTagFail returns a string stating that the validation for "required" tag failed.
func RequiredTagFail(cause string) error {
	return errors.Errorf(requiredTagFailTemplate, cause)
}

// FileTagFail returns a string stating that the validation for "file" tag failed.
func FileTagFail(cause string, value interface{}) error {
	return errors.Errorf(fileTagFailTemplate, cause, value)
}

// URLTagFail returns a string stating that the validation for "url" tag failed.
func URLTagFail(cause string, value interface{}) error {
	return errors.Errorf(urlTagFailTemplate, cause, value)
}

// OneofTagFail returns a string stating that the validation for "oneof" tag failed.
func OneofTagFail(cause string, value interface{}) error {
	return errors.Errorf(oneofTagFailTemplate, cause, value)
}

// HostnamePortTagFail returns a string stating that the validation for "hostname_port" tag failed.
func HostnamePortTagFail(cause string, value interface{}) error {
	return errors.Errorf(hostnamePortTagFailTemplate, cause, value)
}

// MinTagFail returns a string stating that the validation for "min" tag failed.
func MinTagFail(cause string, value interface{}, expValue string) error {
	return errors.Errorf(minTagFailTemplate, cause, value, expValue)
}

// MaxTagFail returns a string stating that the validation for "max" tag failed.
func MaxTagFail(cause string, value interface{}, expValue string) error {
	return errors.Errorf(maxTagFailTemplate, cause, value, expValue)
}

// EqTagFail returns a string stating that the validation for "eq" tag failed.
func EqTagFail(cause string, value interface{}) error {
	return errors.Errorf(eqFailTemplate, cause, value)
}

// LteTagFail returns a string stating that the validation for "lte" tag failed.
func LteTagFail(cause string, value interface{}, expValue string) error {
	return errors.Errorf(lteFailTemplate, cause, value, expValue)
}

// GteTagFail returns a string stating that the validation for "gte" tag failed.
func GteTagFail(cause string, value interface{}, expValue string) error {
	return errors.Errorf(gteFailTemplate, cause, value, expValue)
}

func GtFieldTagFail(cause string, value interface{}) error {
	return errors.Errorf(gtFieldTagFailTemplate, cause, value)
}
