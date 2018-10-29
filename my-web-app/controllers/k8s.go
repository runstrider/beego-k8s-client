package controllers

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/astaxie/beego"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"log"
	"time"
)

var (
	kubeconfig = flag.String("kubeconfig", "/home/ubuntu/.kube/config", "abs path to the k conf")
	clientSet  kubernetes.Clientset
	caStore    cache.Store
	cliConf    *rest.Config
)

type K8sController struct {
	beego.Controller
}

func deployRollBack(nameSpace, dpName string) error {
	deployment, err := clientSet.ExtensionsV1beta1().Deployments(nameSpace).Get(dpName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("could not get deployment info: %v", err)
	}

}

//pod中第一个容器中执行命令
func podExecCMD(podName, podNamespace string, command string) (string, error) {
	var (
		execOut bytes.Buffer
		execErr bytes.Buffer
	)

	pod, err := clientSet.Core().Pods(podNamespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("could not get pod info: %v", err)
	}

	if len(pod.Spec.Containers) == 0 {
		return "", fmt.Errorf("could not find container to exec to")
	}

	req := clientSet.RESTClient().Post().Resource("pods").Name(podName).Namespace(podNamespace).
		SubResource("exec").
		Param("container", pod.Spec.Containers[0].Name).
		Param("command", command).
		//Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("tty", "true")

	exec, err := remotecommand.NewSPDYExecutor(cliConf, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to init exector: %v", err)
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stderr: &execErr,
		Stdout: &execOut,
		Tty:    true,
	})

	if err != nil {
		return "", fmt.Errorf("conld not execute : %v", err)
	}

	if execErr.Len() > 0 {
		return "", fmt.Errorf("stderr: %v", execErr.String())
	}

	return execOut.String(), nil
}

func createPod() {
	var r v1.ResourceRequirements
	j := `{"limits":{"cup":"2000m", "memory":"1Gi"}, "requests":{"cup":"2000m", "memory":"1Gi"}`
	json.Unmarshal([]byte(j), &r)

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "yangdong1",
			Labels: map[string]string{
				"app": "yangdong1",
			},
		},
		Spec: v1.PodSpec{Containers: []v1.Container{{Name: "yangdongcontainer", Image: "ubuntu"}}},
	}
	fmt.Println("Creating pod...")
	podRet, err := clientSet.CoreV1().Pods("kube-system").Create(pod)
	if err != nil {
		panic(err)
	}
	fmt.Println("we had create pod name", podRet.Name)
	podExecCMD(pod.Name, pod.Namespace, "ping www.baidu.com")
}

func (c *K8sController) Get() {
	/*
		deployment, err := clientSet.AppsV1beta1().Deployments("default").Get("yangdong1", metav1.GetOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(deployment)
		c.Ctx.ResponseWriter.Write([]byte(deployment.Name))
	*/
	//create pod "yangdong1"
	createPod()
	c.Ctx.ResponseWriter.Write([]byte("success yangdong1 create"))
}

/*获取访问k8s api客户端*/
func getClient(confPath string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if confPath == "" {
		log.Println("Using in cluster config")
		config, err = rest.InClusterConfig()
	} else {
		log.Println("Using out of cluster config")
		config, err = clientcmd.BuildConfigFromFlags("", confPath)
	}
	if err != nil {
		return nil, err
	}
	cliConf = config
	return kubernetes.NewForConfig(config)
}

func nodeController() {
	watchlist := cache.NewListWatchFromClient(clientSet.Core().RESTClient(), "pods", v1.NamespaceAll, fields.Everything())
	//store 本质为缓存cache{}
	store, controller := cache.NewInformer(watchlist, &v1.Pod{}, time.Second*30, cache.ResourceEventHandlerFuncs{
		AddFunc:    handlePodAdd,
		UpdateFunc: handlePodUpdate,
	},
	)

	caStore = store
	_, exists, err := caStore.GetByKey("yangdong1")
	if exists && err == nil {
		fmt.Println("YANGDONG1 pod exist")
	}
	stop := make(chan struct{})
	go controller.Run(stop)
}

func handlePodAdd(obj interface{}) {
	pod := obj.(*v1.Pod)
	fmt.Println("addingg pod name is", pod.Name)
}

func handlePodUpdate(oldObj, newObj interface{}) {
	opod := oldObj.(*v1.Pod)
	npod := newObj.(*v1.Pod)
	fmt.Printf("updating pod name is %s to %s", opod.Name, npod.Name)
	fmt.Println(caStore.ListKeys())
}

func init() {
	clientset, err := getClient(*kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	clientSet = *clientset

	nodeController()
	//test
	a, _ := clientset.Discovery().ServerGroups()
	fmt.Println("a;sldkfj", a)

}
