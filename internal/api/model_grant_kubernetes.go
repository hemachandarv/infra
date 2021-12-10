/*
Infra API

Infra REST API

API version: 0.1.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package api

import (
	"encoding/json"
)

// GrantKubernetes struct for GrantKubernetes
type GrantKubernetes struct {
	Kind      GrantKubernetesKind `json:"kind"`
	Name      string              `json:"name"`
	Namespace string              `json:"namespace"`
}

// NewGrantKubernetes instantiates a new GrantKubernetes object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewGrantKubernetes(kind GrantKubernetesKind, name string, namespace string) *GrantKubernetes {
	this := GrantKubernetes{}
	this.Kind = kind
	this.Name = name
	this.Namespace = namespace
	return &this
}

// NewGrantKubernetesWithDefaults instantiates a new GrantKubernetes object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewGrantKubernetesWithDefaults() *GrantKubernetes {
	this := GrantKubernetes{}
	return &this
}

// GetKind returns the Kind field value
func (o *GrantKubernetes) GetKind() GrantKubernetesKind {
	if o == nil {
		var ret GrantKubernetesKind
		return ret
	}

	return o.Kind
}

// GetKindOK returns a tuple with the Kind field value
// and a boolean to check if the value has been set.
func (o *GrantKubernetes) GetKindOK() (*GrantKubernetesKind, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Kind, true
}

// SetKind sets field value
func (o *GrantKubernetes) SetKind(v GrantKubernetesKind) {
	o.Kind = v
}

// GetName returns the Name field value
func (o *GrantKubernetes) GetName() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Name
}

// GetNameOK returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *GrantKubernetes) GetNameOK() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value
func (o *GrantKubernetes) SetName(v string) {
	o.Name = v
}

// GetNamespace returns the Namespace field value
func (o *GrantKubernetes) GetNamespace() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Namespace
}

// GetNamespaceOK returns a tuple with the Namespace field value
// and a boolean to check if the value has been set.
func (o *GrantKubernetes) GetNamespaceOK() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Namespace, true
}

// SetNamespace sets field value
func (o *GrantKubernetes) SetNamespace(v string) {
	o.Namespace = v
}

func (o GrantKubernetes) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["kind"] = o.Kind
	}
	if true {
		toSerialize["name"] = o.Name
	}
	if true {
		toSerialize["namespace"] = o.Namespace
	}
	return json.Marshal(toSerialize)
}

type NullableGrantKubernetes struct {
	value *GrantKubernetes
	isSet bool
}

func (v NullableGrantKubernetes) Get() *GrantKubernetes {
	return v.value
}

func (v *NullableGrantKubernetes) Set(val *GrantKubernetes) {
	v.value = val
	v.isSet = true
}

func (v NullableGrantKubernetes) IsSet() bool {
	return v.isSet
}

func (v *NullableGrantKubernetes) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableGrantKubernetes(val *GrantKubernetes) *NullableGrantKubernetes {
	return &NullableGrantKubernetes{value: val, isSet: true}
}

func (v NullableGrantKubernetes) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableGrantKubernetes) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}