package discovery

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	KubernetesProviderName = "kubernetes"
	LocatorProviderName    = "locator"
)

type discoveryProvider interface {
	discoveryServiceUrl(serviceName string) ([]string, error)
}

type DiscoveryService interface {
	FindUrl(serviceName string) (string, error)
}

type DiscoveryServiceImpl struct {
	providers map[string]discoveryProvider
}

type DiscoveryServiceConfig struct {
	// Cache lifetime for founded in locator urls. If duration = 0, urls not caching
	LocatorProviderUrlCacheTime time.Duration

	// Cache lifetime for founded in locator urls. If duration = 0, urls not caching
	KubernetesProviderUrlCacheTime time.Duration

	// Cache lifetime for k8s namespace where pods and services will be found. If duration = 0, urls not caching
	KubernetesProviderNamespaceCacheTime time.Duration

	// Label name for filtering pods by. By default equals "app"
	KubernetesProviderLabelName string
}

func NewDiscoveryService(config *DiscoveryServiceConfig) *DiscoveryServiceImpl {
	kubernetes := newKubernetesDiscoveryProvider(config.KubernetesProviderUrlCacheTime,
		config.KubernetesProviderNamespaceCacheTime,
		config.KubernetesProviderLabelName)

	locator := newLocatorDiscoveryProvider(config.LocatorProviderUrlCacheTime)

	providers := map[string]discoveryProvider{
		KubernetesProviderName: kubernetes,
		LocatorProviderName:    locator,
	}

	return &DiscoveryServiceImpl{providers: providers}
}

func (ds *DiscoveryServiceImpl) getDiscoveryProvider(providerName string) (discoveryProvider, error) {
	provider, ok := ds.providers[providerName]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Discovery provider '%s' not found", providerName))
	}

	return provider, nil
}

func (ds *DiscoveryServiceImpl) FindUrl(serviceName string) (string, error) {
	var urls []string
	for _, p := range ds.providers {
		fundedUrls, err := p.discoveryServiceUrl(serviceName)
		if err != nil {
			log.Println(fmt.Sprintf("Error in discovery: '%s'", err))
			continue
		}
		urls = append(urls, fundedUrls...)
	}

	return balanced(urls)
}

func balanced(urls []string) (string, error) {
	if len(urls) == 0 {
		return "", errors.New("not found in Discovery")
	}
	// todo
	url := strings.TrimSuffix(urls[0], "/")

	return url, nil
}
