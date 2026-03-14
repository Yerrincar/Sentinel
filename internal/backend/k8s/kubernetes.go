package kubernetes

import (
	"context"
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

func GetMetricsFromPod(podName, namespace string) model.ServiceRuntime {
	var s model.ServiceRuntime
	if podName == "" || namespace == "" {
		s.ErrorMsg = "pod name or namespace is empty"
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

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		s.ErrorMsg = err.Error()
		return s
	}
	s.Status = string(pod.Status.Phase)
	if s.Status != "" {
		s.Status = strings.ToUpper(s.Status[:1]) + s.Status[1:]
	}
	s.State = mapPodStatus(pod)
	if pod.Status.StartTime != nil {
		s.Uptime = helpers.FormatUptime(time.Since(pod.Status.StartTime.Time))
	}

	podContainers := pod.Spec.Containers
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

	metricsClient, err := k8sMetrics.NewForConfig(config)
	if err != nil {
		s.ErrorMsg = err.Error()
		return s
	}

	metrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(context.Background(), podName, metav1.GetOptions{})
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
	s.Cpu = defaultSampler.cpuPercent(namespace+"/"+podName, totalCpuNano, time.Now())
	s.Mem = helpers.FormatBytes(totalMemUsage)

	return s
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
