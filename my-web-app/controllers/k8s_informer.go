//自定义crd，并创建informer
package controllers

import (
	"k8s.io/client-go/kubernetes"
	"time"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

//Build custom resouse
type CustomResource struct{
	Name string
	Plural string
	Group string
	version string
	// Scope of the CRD. Namespaced or cluster
	Scope v1beta1.ResourceScope

	// Kind is the serialized interface of the resource.
	Kind string

	ShortNames []string

}

//client's context.Hold the clientsets used for creating and watching custom resouces
type Context struct {
	Clientset kubernetes.Interface
	APIExtensionClientset clientset.Interface
	Interval time.Duration
	TImeout time.Duration
}

func createCrd(context Context) error {

}
