package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"
	"sentinel/internal/model"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func GetMetricsFromPod(podName, namespace string) model.ServiceRuntime {
	var s model.ServiceRuntime
	config, err := getExternalClusterConfig()
	if err != nil {
		panic(fmt.Errorf("failed to get external cluster config: %v", err))
	}
	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		panic(fmt.Errorf("failed to create clientset: %v", err))
	}
	pod, _ := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	s.Status = string(pod.Status.Phase)
	return s
}

func getExternalClusterConfig() (*rest.Config, error) {
	var kubeconfig string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %v", err)
	}

	return config, nil
}
