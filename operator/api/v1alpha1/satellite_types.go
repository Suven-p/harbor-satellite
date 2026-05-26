/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SatelliteSpec defines the desired state of Satellite
type SatelliteSpec struct {

	// Ground Control group names for this satellite. These groups should exist in ground control.
	// +optional
	Groups []string `json:"groups"`

	// Ground Control configuration name for this satellite. This should exist in ground control.
	// +optional
	Config string `json:"config"`

	// Ground Control URL for this satellite.
	// +required
	GroundControlURL string `json:"groundControlURL"`

	// Harbor URL used by Ground Control.
	// +required
	HarborURL string `json:"harborURL"`
}

type RegistrationStatus string

const (
	RegistrationStatusPending    RegistrationStatus = "Pending"
	RegistrationStatusRegistered RegistrationStatus = "Registered"
	RegistrationStatusFailed     RegistrationStatus = "Failed"
)

type SyncStatus string

const (
	SyncStatusInProgress SyncStatus = "InProgress"
	SyncStatusSuccessful SyncStatus = "Successful"
	SyncStatusFailed     SyncStatus = "Failed"
)

// SatelliteStatus defines the observed state of Satellite.
type SatelliteStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Satellite resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Satellite is the Schema for the satellites API
type Satellite struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Satellite
	// +required
	Spec SatelliteSpec `json:"spec"`

	// status defines the observed state of Satellite
	// +optional
	Status SatelliteStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// SatelliteList contains a list of Satellite
type SatelliteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Satellite `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Satellite{}, &SatelliteList{})
}
