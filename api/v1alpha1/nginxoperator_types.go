/*
Copyright 2022.
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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
const (
	ReasonCRNotAvailable          = "OperatorResourceNotAvailable"
	ReasonDeploymentNotAvailable  = "OperandDeploymentNotAvailable"
	ReasonOperandDeploymentFailed = "OperandDeploymentFailed"
	ReasonSucceeded               = "OperatorSucceeded"
)

// NginxOperatorSpec defines the desired state of NginxOperator
type NginxOperatorSpec struct {

	// Port is the port number to expose on the Nginx Pod
	Port *int32 `json:"port,omitempty"`
	// Replicas is the number of deployment replicas to scale
	Replicas *int32 `json:"replicas,omitempty"`
	// ForceRedploy is any string, modifying this field
	// instructs the Operator to redeploy the Operand
	ForceRedploy string `json:"forceRedploy,omitempty"`
	//AppLimitCost is the limit for the deployment of an app
	AppLimitCost *int32 `json:"appLimitCost,omitempty"`
	//IsDeployed indicates if the app is deployed on this cluster
	IsDeployed bool `json:"isDeployed,omitempty"`
}

// NginxOperatorStatus defines the observed state of NginxOperator
type NginxOperatorStatus struct {

	// Conditions is the list of status condition updates
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NginxOperator is the Schema for the nginxoperators API
type NginxOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NginxOperatorSpec   `json:"spec,omitempty"`
	Status NginxOperatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NginxOperatorList contains a list of NginxOperator
type NginxOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NginxOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NginxOperator{}, &NginxOperatorList{})
}
