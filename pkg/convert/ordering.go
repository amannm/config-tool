package convert

import (
	"reflect"
	"strings"
)

type OrderKey struct {
	key        string
	valueOrder []string
}

func testKeyValueMatch(query JSONObject, target JSONObject) bool {
	for k, v := range query {
		if targetValue, ok := target[k]; ok {
			if !reflect.DeepEqual(targetValue, v) {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

const setElementOrderPrefix = "$setElementOrder/"

func ExecutePatchOrdering(o JSONObject) (JSONObject, error) {
	ordering := map[string][]JSONObject{}
	result := JSONObject{}
	for k, v := range o {
		switch typedValue := v.(type) {
		case JSONObject:
			reorderedValue, err := ExecutePatchOrdering(typedValue)
			if err != nil {
				return nil, err
			}
			result[k] = reorderedValue
		case JSONArray:
			if strings.HasPrefix(k, setElementOrderPrefix) {
				nameOrdering := []JSONObject{}
				if nameList, ok := v.(JSONArray); ok {
					for _, item := range nameList {
						if typedItem, ok := item.(JSONObject); ok {
							nameOrdering = append(nameOrdering, typedItem)
						}
					}
				}
				targetKey := strings.TrimPrefix(k, setElementOrderPrefix)
				ordering[targetKey] = nameOrdering
			} else {
				if nameOrdering, ok := ordering[k]; ok {
					reordered := []JSONObject{}
					for _, nameOrder := range nameOrdering {
						for _, valueToOrder := range typedValue {
							if typedValueToOrder, ok := valueToOrder.(JSONObject); ok {
								if testKeyValueMatch(nameOrder, typedValueToOrder) {
									reordered = append(reordered, typedValueToOrder)
								}
							}
						}
					}
					result[k] = reordered
				} else {
					for _, valueItem := range typedValue {
						if typedValueItem, ok := valueItem.(JSONObject); ok {
							reorderedValueItem, err := ExecutePatchOrdering(typedValueItem)
							if err != nil {
								return nil, err
							}
							result[k] = reorderedValueItem
						} else {
							result[k] = valueItem
						}
					}
				}
			}
		default:
			result[k] = v
		}
	}
	return result, nil
}
