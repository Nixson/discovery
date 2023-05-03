package discovery

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	envServiceUrlCachePrefix = "ENV_SERVICE_URL_CACHE_"
)

type locatorDiscoveryProviderImpl struct {
	urlCache *cache.Cache
}

func newLocatorDiscoveryProvider(urlCacheTime time.Duration) *locatorDiscoveryProviderImpl {
	var result locatorDiscoveryProviderImpl

	if urlCacheTime > 0 {
		result.urlCache = cache.New(urlCacheTime, urlCacheTime)
	}

	return &result
}

func (d *locatorDiscoveryProviderImpl) discoveryServiceUrl(serviceName string) ([]string, error) {
	return d.getServiceLocator(serviceName)
}

func (d *locatorDiscoveryProviderImpl) getServiceLocator(serviceName string) ([]string, error) {
	cachedServiceUrls := d.getUrlCache(serviceName)
	if len(cachedServiceUrls) > 0 {
		return cachedServiceUrls, nil
	}

	serviceLocatorStr := os.Getenv("SERVICELOCATOR_" + serviceName)
	result := strings.Split(serviceLocatorStr, ",")
	if len(result) == 0 {
		return []string{}, errors.New("service not found in Locator Discovery")
	}

	d.setUrlCache(serviceName, result)
	return result, nil
}

func (d *locatorDiscoveryProviderImpl) getUrlCache(serviceName string) []string {
	if d.urlCache == nil {
		return []string{}
	}

	cachedServiceUrl, _ := d.urlCache.Get(getServiceUrlCacheKey(envServiceUrlCachePrefix, serviceName))
	return cachedServiceUrl.([]string)
}

func (d *locatorDiscoveryProviderImpl) setUrlCache(serviceName string, serviceUrls []string) {
	if d.urlCache == nil {
		return
	}

	d.urlCache.Set(getServiceUrlCacheKey(envServiceUrlCachePrefix, serviceName), serviceUrls, 0)
}
