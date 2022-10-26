/*
 * CLOUD API
 *
 * An enterprise-grade Infrastructure is provided as a Service (IaaS) solution that can be managed through a browser-based \"Data Center Designer\" (DCD) tool or via an easy to use API.   The API allows you to perform a variety of management tasks such as spinning up additional servers, adding volumes, adjusting networking, and so forth. It is designed to allow users to leverage the same power and flexibility found within the DCD visual tool. Both tools are consistent with their concepts and lend well to making the experience smooth and intuitive.
 *
 * API version: 5.0
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package ionossdk

import (
	"encoding/json"
)

// ErrorMessage struct for ErrorMessage
type ErrorMessage struct {
	// Application internal error code
	ErrorCode *string `json:"errorCode,omitempty"`
	// Human readable message
	Message *string `json:"message,omitempty"`
}



// GetErrorCode returns the ErrorCode field value
// If the value is explicit nil, the zero value for string will be returned
func (o *ErrorMessage) GetErrorCode() *string {
	if o == nil {
		return nil
	}

	return o.ErrorCode
}

// GetErrorCodeOk returns a tuple with the ErrorCode field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *ErrorMessage) GetErrorCodeOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.ErrorCode, true
}

// SetErrorCode sets field value
func (o *ErrorMessage) SetErrorCode(v string) {
	o.ErrorCode = &v
}

// HasErrorCode returns a boolean if a field has been set.
func (o *ErrorMessage) HasErrorCode() bool {
	if o != nil && o.ErrorCode != nil {
		return true
	}

	return false
}



// GetMessage returns the Message field value
// If the value is explicit nil, the zero value for string will be returned
func (o *ErrorMessage) GetMessage() *string {
	if o == nil {
		return nil
	}

	return o.Message
}

// GetMessageOk returns a tuple with the Message field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *ErrorMessage) GetMessageOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Message, true
}

// SetMessage sets field value
func (o *ErrorMessage) SetMessage(v string) {
	o.Message = &v
}

// HasMessage returns a boolean if a field has been set.
func (o *ErrorMessage) HasMessage() bool {
	if o != nil && o.Message != nil {
		return true
	}

	return false
}


func (o ErrorMessage) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}

	if o.ErrorCode != nil {
		toSerialize["errorCode"] = o.ErrorCode
	}
	

	if o.Message != nil {
		toSerialize["message"] = o.Message
	}
	
	return json.Marshal(toSerialize)
}

type NullableErrorMessage struct {
	value *ErrorMessage
	isSet bool
}

func (v NullableErrorMessage) Get() *ErrorMessage {
	return v.value
}

func (v *NullableErrorMessage) Set(val *ErrorMessage) {
	v.value = val
	v.isSet = true
}

func (v NullableErrorMessage) IsSet() bool {
	return v.isSet
}

func (v *NullableErrorMessage) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableErrorMessage(val *ErrorMessage) *NullableErrorMessage {
	return &NullableErrorMessage{value: val, isSet: true}
}

func (v NullableErrorMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableErrorMessage) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


