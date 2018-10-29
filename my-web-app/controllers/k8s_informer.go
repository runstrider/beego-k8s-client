//The informer is relate with wacher folder.
package controllers

import (
	"fmt"
	"os"
	"beego/my-web-app/controllers/watcher"
	"os/signal"
	"syscall"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"time"
)

var kubeconfigtmp = "/home/ubunut/.kube/config"

func start() {
	context, clientSet, err := createContext(kubeconfigtmp)
	if err != nil{
		fmt.Printf("Failed to create context. %+v\n", err)
		os.Exit(1)
	}

	//Create crd resources
	resources := []watcher.CustomResource{}
	err = watcher.CreateCustomResources(*context, resources)
	if err != nil{
		fmt.Printf("Failed to create custom resource. %v\n", err)
		os.Exit(1)
	}

	//创建stop watcher channel
	signalChan := make(chan os.Signal, 1)
	stopChan := make(chan struct{})
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	controller := new

	for {
		select {
		case <-signalChan:
			fmt.Println("Bye bye!")
			close(stopChan)
			return
		}
	}
}

func createContext(kubeconfig string)(watcher.Context, err){
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil{
		return nil, nil, fmt.Errorf("Failed to get k8s config. %+v", err)
	}

	clientset,err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil,nil, fmt.Errorf("Failed to create k8s client. %+v", err)
	}

	apiExtClientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil{
		return nil,nil, fmt.Errorf("Failed to create k8s apiextension client. %+v", err)
	}


	context := watcher.Context{
		Clientset:clientset,
		APIExtensionClientset:apiExtClientset,
		Interval:100*time.Millisecond,
		TImeout:60*time.Second,
	}
	return context,,nil
}