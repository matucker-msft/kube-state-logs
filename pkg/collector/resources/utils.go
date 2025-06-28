package resources

import (
	"reflect"
	"strings"

	"k8s.io/client-go/tools/cache"
)

// convertStructToMap is a generic function that converts any struct to map[string]any using reflection
func convertStructToMap(data any) map[string]any {
	result := make(map[string]any)

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return result
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Get JSON tag name
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Remove comma and options from JSON tag
		if commaIndex := strings.Index(jsonTag, ","); commaIndex != -1 {
			jsonTag = jsonTag[:commaIndex]
		}

		// Convert field value to any
		var value any
		switch field.Kind() {
		case reflect.Ptr:
			if field.IsNil() {
				value = nil
			} else {
				value = field.Elem().Interface()
			}
		default:
			value = field.Interface()
		}

		result[jsonTag] = value
	}

	return result
}

// safeGetStoreList safely gets the list from an informer store with nil checks
func safeGetStoreList(informer cache.SharedIndexInformer) []any {
	if informer == nil {
		return []any{}
	}

	store := informer.GetStore()
	if store == nil {
		return []any{}
	}

	return store.List()
}
