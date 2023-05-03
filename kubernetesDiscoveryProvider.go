package discovery

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuber "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	currentNameSpaceCacheKey = "CURRENT_NAME_SPACE_CACHE"
	serviceUrlCachePrefix    = "SERVICE_URL_CACHE_"
	defaultLabelName         = "app"
)

type kubernetesDiscoveryProviderImpl struct {
	namespaceCache *cache.Cache
	urlCache       *cache.Cache
	labelName      string
}

func newKubernetesDiscoveryProvider(urlCacheTime time.Duration, namespaceCacheTime time.Duration, labelName string) *kubernetesDiscoveryProviderImpl {
	result := kubernetesDiscoveryProviderImpl{
		labelName: defaultLabelName,
	}

	if urlCacheTime > 0 {
		result.urlCache = cache.New(urlCacheTime, urlCacheTime)
	}

	if namespaceCacheTime > 0 {
		result.namespaceCache = cache.New(namespaceCacheTime, namespaceCacheTime)
	}

	if labelName != "" {
		result.labelName = labelName
	}

	return &result
}

func (d *kubernetesDiscoveryProviderImpl) discoveryServiceUrl(serviceName string) ([]string, error) {
	cachedServiceUrl := d.getUrlCache(serviceName)
	if cachedServiceUrl != "" {
		return []string{cachedServiceUrl}, nil
	}

	var currentNamespace string

	if cachedCurrentNamespace := d.getNamespaceCache(); cachedCurrentNamespace != "" {
		currentNamespace = cachedCurrentNamespace
	} else {
		currentNamespaceBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			fmt.Println("Kubernetes Server not available")
			return []string{}, err
		}

		currentNamespace = string(currentNamespaceBytes)
		d.setNamespaceCache(currentNamespace)
	}

	fmt.Println("Current kubernetes namespace: " + currentNamespace)

	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err)
		return []string{}, err
	}
	// creates the clientset
	clientset, err := kuber.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
		return []string{}, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%v=%v", d.labelName, serviceName),
	}

	pods, err := clientset.CoreV1().Pods(currentNamespace).List(context.TODO(), listOptions)
	if err != nil {
		fmt.Println(err)
		return []string{}, err
	}

	var existLabeledPods []string

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				existLabeledPods = append(existLabeledPods, pod.Name)
			}
		}
	}

	if len(existLabeledPods) == 0 {
		return []string{}, err
	}

	services, err := clientset.CoreV1().Services(currentNamespace).List(context.TODO(), listOptions)
	if err != nil {
		fmt.Println(err)
		return []string{}, err
	}

	for _, v := range services.Items {
		serviceUrl := fmt.Sprintf("%s://%s:%d", v.Spec.Ports[0].Name, v.Spec.ClusterIP, v.Spec.Ports[0].Port)
		d.setUrlCache(serviceName, serviceUrl)

		return []string{serviceUrl}, nil
	}

	return []string{}, errors.New("service not found in Kubernetes Discovery")
}

func getServiceUrlCacheKey(cachePrefix, serviceName string) string {
	return fmt.Sprintf("%v%v", cachePrefix, serviceName)
}

func (d *kubernetesDiscoveryProviderImpl) getNamespaceCache() string {
	if d.namespaceCache == nil {
		return ""
	}

	if cachedCurrentNamespace, ok := d.namespaceCache.Get(currentNameSpaceCacheKey); ok {
		return cachedCurrentNamespace.(string)
	}
	return ""
}

func (d *kubernetesDiscoveryProviderImpl) setNamespaceCache(namespace string) {
	if d.namespaceCache == nil {
		return
	}

	d.namespaceCache.Set(currentNameSpaceCacheKey, namespace, 0)
}

func (d *kubernetesDiscoveryProviderImpl) getUrlCache(serviceName string) string {
	if d.urlCache == nil {
		return ""
	}

	if cachedServiceUrl, ok := d.urlCache.Get(getServiceUrlCacheKey(serviceUrlCachePrefix, serviceName)); ok {
		fmt.Println(cachedServiceUrl)
		return cachedServiceUrl.(string)
	}
	return ""
}

func (d *kubernetesDiscoveryProviderImpl) setUrlCache(serviceName, serviceUrl string) {
	if d.urlCache == nil {
		return
	}

	d.urlCache.Set(getServiceUrlCacheKey(serviceUrlCachePrefix, serviceName), serviceUrl, 0)
}
