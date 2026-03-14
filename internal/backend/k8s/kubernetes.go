package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sentinel/internal/model"
	helpers "sentinel/internal/util"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	k8sMetrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type cpuSample struct {
	usageUsec uint64
	at        time.Time
}

type Sampler struct {
	prevByID map[string]cpuSample
}

var defaultSampler = NewSampler()

func NewSampler() *Sampler {
	return &Sampler{prevByID: map[string]cpuSample{}}
}

func GetMetricsFromDeployment(deploymentName, namespace string) model.ServiceRuntime {
	var s model.ServiceRuntime
	if deploymentName == "" || namespace == "" {
		s.ErrorMsg = "deployment name or namespace is empty"
		return s
	}

	config, err := getExternalClusterConfig()
	if err != nil {
		s.ErrorMsg = err.Error()
		return s
	}
	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		s.ErrorMsg = fmt.Sprintf("failed to create clientset: %v", err)
		return s
	}

	ctx := context.Background()
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		s.ErrorMsg = err.Error()
		return s
	}

	replicas := int32(1)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	podContainers := deployment.Spec.Template.Spec.Containers
	totalLimit := uint64(0)
	hasMemoryLimit := true
	for _, p := range podContainers {
		memLimit, ok := p.Resources.Limits[corev1.ResourceMemory]
		if !ok {
			hasMemoryLimit = false
			continue
		}
		if v := memLimit.Value(); v > 0 {
			totalLimit += uint64(v)
		}
	}
	if !hasMemoryLimit || totalLimit == 0 {
		s.MemLimit = "No limit assigned"
	} else {
		s.MemLimit = helpers.FormatBytes(totalLimit)
	}

	if replicas == 0 {
		s.Status = "Inactive"
		s.State = "inactive"
		s.Cpu = 0
		s.Mem = "0 B"
		s.Uptime = "0s"
		return s
	}

	if deployment.Spec.Selector == nil {
		s.ErrorMsg = "deployment selector is empty"
		return s
	}
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		s.ErrorMsg = err.Error()
		return s
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		s.ErrorMsg = err.Error()
		return s
	}
	if len(pods.Items) == 0 {
		s.Status = "Pending"
		s.State = "degraded"
		s.ErrorMsg = "no pods found for deployment"
		return s
	}
	pod := pickDeploymentPod(pods.Items)

	s.Status = string(pod.Status.Phase)
	if s.Status != "" {
		s.Status = strings.ToUpper(s.Status[:1]) + s.Status[1:]
	}

	s.State = mapPodStatus(&pod)
	if pod.Status.StartTime != nil {
		s.Uptime = helpers.FormatUptime(time.Since(pod.Status.StartTime.Time))
	}

	metricsClient, err := k8sMetrics.NewForConfig(config)
	if err != nil {
		s.ErrorMsg = err.Error()
		return s
	}

	metrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		s.ErrorMsg = err.Error()
		return s
	}

	containersMetrics := metrics.Containers
	var totalCpuNano int64
	var totalMemUsage uint64
	for _, c := range containersMetrics {
		totalCpuNano += c.Usage.Cpu().ScaledValue(resource.Nano)
		if v := c.Usage.Memory().Value(); v > 0 {
			totalMemUsage += uint64(v)
		}
	}
	s.Cpu = defaultSampler.cpuPercent(namespace+"/"+deploymentName, totalCpuNano, time.Now())
	s.Mem = helpers.FormatBytes(totalMemUsage)

	return s
}

func GetDeployment(deploymentName, namespace string) error {
	ctx := context.Background()
	config, err := getExternalClusterConfig()
	if err != nil {
		return err
	}
	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		return err
	}
	_, err = clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return nil
}

func K8sStart(namespace, deploymentName string) error {
	ctx := context.Background()
	config, err := getExternalClusterConfig()
	if err != nil {
		return err
	}
	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		return err
	}

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	replicas := int32(1)
	deployment.Spec.Replicas = &replicas
	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func K8sStop(namespace, deploymentName string) error {
	ctx := context.Background()
	config, err := getExternalClusterConfig()
	if err != nil {
		return err
	}
	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		return err
	}

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	replicas := int32(0)
	deployment.Spec.Replicas = &replicas
	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func K8sRestart(namespace, deploymentName string) error {
	ctx := context.Background()
	config, err := getExternalClusterConfig()
	if err != nil {
		return err
	}
	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		return err
	}

	restartedAt := time.Now().Format(time.RFC3339)
	patch := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{
						"kubectl.kubernetes.io/restartedAt": restartedAt,
					},
				},
			},
		},
	}
	body, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = clientset.AppsV1().Deployments(namespace).Patch(ctx, deploymentName, k8stypes.StrategicMergePatchType, body, metav1.PatchOptions{})
	return err
}

func pickDeploymentPod(pods []corev1.Pod) corev1.Pod {
	for _, p := range pods {
		if p.Status.Phase == corev1.PodRunning {
			return p
		}
	}
	return pods[0]
}

func (s *Sampler) cpuPercent(serviceID string, usageNano int64, now time.Time) float64 {
	if s.prevByID == nil {
		s.prevByID = make(map[string]cpuSample)
	}
	prev, ok := s.prevByID[serviceID]
	if !ok || usageNano < 0 {
		s.prevByID[serviceID] = cpuSample{usageUsec: 0, at: now}
		return 0.0
	}

	dt := now.Sub(prev.at).Microseconds()
	if dt <= 0 {
		return 0.0
	}

	du := uint64((float64(usageNano) / 1_000_000_000.0) * float64(dt))
	curr := prev.usageUsec + du
	s.prevByID[serviceID] = cpuSample{usageUsec: curr, at: now}

	return helpers.CPUPercent(prev.usageUsec, curr, prev.at, now)
}

func getExternalClusterConfig() (*rest.Config, error) {
	candidates := make([]string, 0, 4)
	seen := map[string]struct{}{}

	add := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		candidates = append(candidates, p)
	}

	if envKC := strings.TrimSpace(os.Getenv("KUBECONFIG")); envKC != "" {
		for _, p := range filepath.SplitList(envKC) {
			add(strings.TrimSpace(p))
		}
	}

	if sudoUser := strings.TrimSpace(os.Getenv("SUDO_USER")); sudoUser != "" {
		if u, err := user.Lookup(sudoUser); err == nil && u.HomeDir != "" {
			add(filepath.Join(u.HomeDir, ".kube", "config"))
		} else {
			add(filepath.Join("/home", sudoUser, ".kube", "config"))
		}
	}

	if home := homedir.HomeDir(); home != "" {
		add(filepath.Join(home, ".kube", "config"))
	}

	for _, kc := range candidates {
		if _, err := os.Stat(kc); err != nil {
			continue
		}
		cfg, err := clientcmd.BuildConfigFromFlags("", kc)
		if err == nil {
			return cfg, nil
		}
	}

	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}

	return nil, fmt.Errorf("failed to build k8s config; checked: %v", candidates)
}

func mapPodStatus(pod *corev1.Pod) string {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				return "degraded"
			}
		}
		return "running"
	case corev1.PodPending, corev1.PodUnknown:
		return "degraded"
	case corev1.PodSucceeded, corev1.PodFailed:
		return "stopped"
	default:
		return "degraded"
	}
}

func UpdateEnvKubeconfig(newValue, key string) error {
	input, err := os.ReadFile("./.env")
	if err != nil {
		return err
	}
	lines := strings.Split(string(input), "\n")
	found := false
	entry := fmt.Sprintf("%s=%s", key, newValue)

	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = entry
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, entry)
	}

	output := strings.Join(lines, "\n")
	return os.WriteFile("./.env", []byte(output), 0644)
}
