package resources

import (
	"github.com/matucker-msft/kube-state-logs/pkg/utils"
	"k8s.io/client-go/tools/cache"
)

// convertStructToMap is a backward compatibility alias for utils.ConvertStructToMap
func convertStructToMap(data any) map[string]any {
	return utils.ConvertStructToMap(data)
}

// safeGetStoreList is a backward compatibility alias for utils.SafeGetStoreList
func safeGetStoreList(informer cache.SharedIndexInformer) []any {
	return utils.SafeGetStoreList(informer)
}
