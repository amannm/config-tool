package convert

import (
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8spatch "k8s.io/apimachinery/pkg/util/strategicpatch"
	"reflect"
	"sigs.k8s.io/yaml"
)

type JSONObject = map[string]any
type JSONArray = []any
type JSONValue = any

type PatchGenerator struct {
	schemaClient *SchemaClient
}

type PatchGenerationResult struct {
	base    JSONObject
	patches []JSONObject
}

func NewPatchGenerator(schemaFolderPath string) (*PatchGenerator, error) {
	sc, err := NewSchemaClient(schemaFolderPath)
	if err != nil {
		return nil, err
	}
	c := &PatchGenerator{
		sc,
	}
	return c, nil
}

func (pgr *PatchGenerationResult) GetBaseYAML() ([]byte, error) {
	baseBytes, err := json.MarshalIndent(pgr.base, "", "    ")
	if err != nil {
		return nil, err
	}
	return yaml.JSONToYAML(baseBytes)
}

func (pgr *PatchGenerationResult) GetPatchYAMLs() ([][]byte, error) {
	result := [][]byte{}
	for _, patch := range pgr.patches {
		patchBytes, err := json.MarshalIndent(patch, "", "    ")
		if err != nil {
			return nil, err
		}
		yamlPatchBytes, err := yaml.JSONToYAML(patchBytes)
		if err != nil {
			return nil, err
		}
		result = append(result, yamlPatchBytes)
	}
	return result, nil
}

func (pg *PatchGenerator) Execute(resources []JSONObject) ([]PatchGenerationResult, error) {

	partitions := map[schema.GroupVersionKind][]JSONObject{}
	for _, resource := range resources {
		gvk, err := ComputeGVK(resource)
		if err != nil {
			return nil, err
		}
		partition, ok := partitions[*gvk]
		if ok {
			partitions[*gvk] = append(partition, resource)
		} else {
			partitions[*gvk] = []JSONObject{resource}
		}
	}
	generationResults := []PatchGenerationResult{}
	for gvk, partition := range partitions {
		patchMeta, err := pg.schemaClient.GetPatchMetadata(gvk)
		if err != nil {
			return nil, err
		}
		base := partition[0]
		for i := 1; i < len(partition); i++ {
			other := partition[i]
			patch, err := calculatePatch(other, base, patchMeta)
			if err != nil {
				return nil, err
			}
			base, err = subtractObject(base, patch, k8spatch.PatchMeta{}, patchMeta)
			if err != nil {
				return nil, err
			}
		}
		result := PatchGenerationResult{
			base:    base,
			patches: []JSONObject{},
		}
		for i := 0; i < len(partition); i++ {
			item := partition[i]
			patch, err := calculatePatch(base, item, patchMeta)
			if err != nil {
				return nil, err
			}
			result.patches = append(result.patches, patch)
		}
		generationResults = append(generationResults, result)
	}
	return generationResults, nil
}

func calculatePatch(content JSONObject, other JSONObject, lookupMeta k8spatch.LookupPatchMeta) (JSONObject, error) {
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
	var patch JSONObject
	err = json.Unmarshal(patchBytes, &patch)
	if err != nil {
		return nil, err
	}
	return patch, nil
}

func subtractObject(a JSONObject, b JSONObject, patchMeta k8spatch.PatchMeta, patchContext k8spatch.LookupPatchMeta) (JSONObject, error) {
	result := JSONObject{}
	for k, aValue := range a {
		bValue, ok := b[k]
		if ok {
			switch typedAValue := aValue.(type) {
			case JSONObject:
				switch typedBValue := bValue.(type) {
				case JSONObject:
					lookupMeta, patchMeta, err := patchContext.LookupPatchMetadataForStruct(k)
					if err != nil {
						return nil, err
					}
					result[k], err = subtractObject(typedAValue, typedBValue, patchMeta, lookupMeta)
					if err != nil {
						return nil, err
					}
					continue
				}
			case JSONArray:
				switch typedBValue := bValue.(type) {
				case JSONArray:
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
				if reflect.DeepEqual(aValue, bValue) && k != patchMeta.GetPatchMergeKey() {
					continue
				}
			}
		}
		result[k] = aValue
	}
	return result, nil
}

func subtractList(a JSONArray, b JSONArray, listPatchMetadata k8spatch.PatchMeta, listSchema k8spatch.LookupPatchMeta) (JSONArray, error) {
	result := JSONArray{}
	for _, aValue := range a {
		switch typedAValue := aValue.(type) {
		case JSONObject:
			subtractedItem, err := subtractObjectListItem(typedAValue, b, listPatchMetadata, listSchema)
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
func subtractObjectListItem(aValue JSONObject, b JSONArray, patchMeta k8spatch.PatchMeta, patchContext k8spatch.LookupPatchMeta) (JSONObject, error) {
	mergeKey := patchMeta.GetPatchMergeKey()
	aMergeValue, ok := aValue[mergeKey]
	if !ok {
		return nil, fmt.Errorf("unexpected missing merge key")
	}
	for _, bValue := range b {
		switch typedBValue := bValue.(type) {
		case JSONObject:
			bMergeValue, ok := typedBValue[mergeKey]
			if !ok {
				return nil, fmt.Errorf("unexpected missing merge key")
			}
			if aMergeValue == bMergeValue {
				subtractedObject, err := subtractObject(aValue, typedBValue, patchMeta, patchContext)
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
func subtractNonObjectListItem(aValue JSONValue, b JSONArray) JSONValue {
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
