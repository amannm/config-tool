package convert

import (
	"encoding/json"
	"fmt"
	k8spatch "k8s.io/apimachinery/pkg/util/strategicpatch"
	"reflect"
	"sigs.k8s.io/yaml"
)

type PatchGenerator struct {
	schemaClient *SchemaClient
}

func NewPatchGenerator(schemaPath string) (*PatchGenerator, error) {
	sc, err := NewSchemaClient(schemaPath)
	if err != nil {
		return nil, err
	}
	c := &PatchGenerator{
		sc,
	}
	return c, nil
}

func (pg *PatchGenerator) ExtractBase(content map[string]interface{}, other map[string]interface{}) ([]byte, error) {
	patchMeta := pg.schemaClient.GetPatchMetadata("io.k8s.api.apps.v1.Deployment")
	patch, err := calculatePatch(content, other, patchMeta)
	if err != nil {
		return nil, err
	}
	base, err := subtractObject(other, patch, patchMeta)
	if err != nil {
		return nil, err
	}
	baseBytes, err := json.MarshalIndent(base, "", "    ")
	if err != nil {
		return nil, err
	}

	yamlPatch, err := yaml.JSONToYAML(baseBytes)
	if err != nil {
		return nil, err
	}
	return yamlPatch, nil
	//patched, err := k8spatch.StrategicMergePatchUsingLookupPatchMeta(nil, nil, patchMeta)

}

func calculatePatch(content map[string]interface{}, other map[string]interface{}, lookupMeta k8spatch.LookupPatchMeta) (map[string]interface{}, error) {
	contentBytes, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	otherBytes, err := json.Marshal(other)
	if err != nil {
		return nil, err
	}
	patchBytes, err := k8spatch.CreateTwoWayMergePatchUsingLookupPatchMeta(contentBytes, otherBytes, lookupMeta)
	if err != nil {
		return nil, err
	}
	var patch map[string]interface{}
	err = json.Unmarshal(patchBytes, &patch)
	if err != nil {
		return nil, err
	}
	return patch, nil
}

func subtractObject(a map[string]interface{}, b map[string]interface{}, patchContext k8spatch.LookupPatchMeta) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for k, aValue := range a {
		bValue, ok := b[k]
		if ok {
			switch typedAValue := aValue.(type) {
			case map[string]interface{}:
				switch typedBValue := bValue.(type) {
				case map[string]interface{}:
					lookupMeta, _, err := patchContext.LookupPatchMetadataForStruct(k)
					if err != nil {
						return nil, err
					}
					result[k], err = subtractObject(typedAValue, typedBValue, lookupMeta)
					if err != nil {
						return nil, err
					}
					continue
				}
			case []interface{}:
				switch typedBValue := bValue.(type) {
				case []interface{}:
					lookupMeta, patchMeta, err := patchContext.LookupPatchMetadataForSlice(k)
					if err != nil {
						return nil, err
					}
					if shouldSubtractList(patchMeta) {
						result[k], err = subtractList(typedAValue, typedBValue, patchMeta, lookupMeta)
					}
					if err != nil {
						return nil, err
					}
					continue
				}
			default:
				if reflect.DeepEqual(aValue, bValue) {
					continue
				}
			}
		}
		result[k] = aValue
	}
	return result, nil
}

func subtractList(a []interface{}, b []interface{}, patchMeta k8spatch.PatchMeta, patchContext k8spatch.LookupPatchMeta) ([]interface{}, error) {
	result := []interface{}{}
	for _, aValue := range a {
		switch typedAValue := aValue.(type) {
		case map[string]interface{}:
			mergeKey := patchMeta.GetPatchMergeKey()
			subtractedItem, err := subtractObjectListItem(typedAValue, b, mergeKey, patchContext)
			if err != nil {
				return nil, err
			}
			if subtractedItem != nil {
				result = append(result, subtractedItem)
			}
		default:
			subtractedItem := subtractNonObjectListItem(typedAValue, b)
			if subtractedItem != nil {
				result = append(result, subtractedItem)
			}
		}
	}
	return result, nil
}
func subtractObjectListItem(aValue map[string]interface{}, b []interface{}, mergeKey string, patchContext k8spatch.LookupPatchMeta) (map[string]interface{}, error) {
	aMergeValue, ok := aValue[mergeKey]
	if !ok {
		return nil, fmt.Errorf("unexpected missing merge key")
	}
	for _, bValue := range b {
		switch typedBValue := bValue.(type) {
		case map[string]interface{}:
			bMergeValue, ok := typedBValue[mergeKey]
			if !ok {
				return nil, fmt.Errorf("unexpected missing merge key")
			}
			if aMergeValue == bMergeValue {
				subtractedObject, err := subtractObject(aValue, typedBValue, patchContext)
				if err != nil {
					return nil, err
				}
				if len(subtractedObject) > 0 {
					return subtractedObject, nil
				} else {
					return nil, nil
				}
			}
		}
	}
	return aValue, nil
}
func subtractNonObjectListItem(aValue interface{}, b []interface{}) interface{} {
	for _, bValue := range b {
		if reflect.DeepEqual(aValue, bValue) {
			return nil
		}
	}
	return aValue
}
func shouldSubtractList(patchMeta k8spatch.PatchMeta) bool {
	strategies := patchMeta.GetPatchStrategies()
	for _, strategy := range strategies {
		if strategy == "merge" {
			return true
		}
	}
	return false
}
