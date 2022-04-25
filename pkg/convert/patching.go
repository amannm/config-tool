package convert

import (
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8spatch "k8s.io/apimachinery/pkg/util/strategicpatch"
	"log"
	"os"
	"path"
	"reflect"
	"sigs.k8s.io/yaml"
)

type JSONObject = map[string]any
type JSONArray = []any
type JSONValue = any

type PatchGenerator struct {
	schemaClient *SchemaClient
}

func (pgr *PatchPartition) String() string {
	jsonContent, _ := json.Marshal(pgr.base)
	yamlContent, _ := yaml.JSONToYAML(jsonContent)
	return fmt.Sprintf("gvk = %s\n\n%s", pgr.gvk, yamlContent)
}

func (pgr *PatchPartition) DumpToFolder(directoryPath string) error {
	rootDir := path.Join(directoryPath, fmt.Sprintf("%s_%s_%s", pgr.gvk.Group, pgr.gvk.Version, pgr.gvk.Kind))
	jsonContent, _ := json.Marshal(pgr.base)
	yamlContent, _ := yaml.JSONToYAML(jsonContent)
	err := os.Mkdir(rootDir, 0755)
	if err != nil {
		return err
	}
	err = WriteFile(yamlContent, path.Join(rootDir, "base.yaml"))
	if err != nil {
		return err
	}
	for _, source := range pgr.sources {
		if len(source.patch) > 0 {
			jsonContent, _ := json.Marshal(source.patch)
			yamlContent, _ := yaml.JSONToYAML(jsonContent)
			err = WriteFile(yamlContent, path.Join(rootDir, fmt.Sprintf("%s.yaml", source.name)))
			if err != nil {
				return err
			}
		}
	}
	return nil
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

func (pgr *PatchPartition) GetBaseYAML() ([]byte, error) {
	baseBytes, err := json.MarshalIndent(pgr.base, "", "    ")
	if err != nil {
		return nil, err
	}
	return yaml.JSONToYAML(baseBytes)
}

func (pgr *PatchPartition) GetPatchYAMLs() ([][]byte, error) {
	result := [][]byte{}
	for _, source := range pgr.sources {
		if len(source.patch) > 0 {
			jsonBytes, err := json.MarshalIndent(source.patch, "", "    ")
			if err != nil {
				return nil, err
			}
			yamlBytes, err := yaml.JSONToYAML(jsonBytes)
			if err != nil {
				return nil, err
			}
			result = append(result, yamlBytes)
		}
	}
	return result, nil
}

func (pgr *PatchPartition) GetOriginalYAMLs() ([][]byte, error) {
	result := [][]byte{}
	for _, source := range pgr.sources {
		jsonBytes, err := json.MarshalIndent(source.original, "", "    ")
		if err != nil {
			return nil, err
		}
		yamlBytes, err := yaml.JSONToYAML(jsonBytes)
		if err != nil {
			return nil, err
		}
		result = append(result, yamlBytes)
	}
	return result, nil
}

type PatchSource struct {
	name     string
	original JSONObject
	patch    JSONObject
}
type PatchPartition struct {
	gvk     schema.GroupVersionKind
	base    JSONObject
	sources []PatchSource
}

func (pg *PatchGenerator) Execute(resources []JSONObject) ([]PatchPartition, error) {
	partitions := map[schema.GroupVersionKind]PatchPartition{}
	for _, resource := range resources {
		gvk, err := ComputeGVK(resource)
		if err != nil {
			return nil, err
		}
		name, err := GetResourceName(resource)
		if err != nil {
			return nil, err
		}
		partition, ok := partitions[*gvk]
		if ok {
			next := append(partition.sources, PatchSource{
				name:     name,
				original: resource,
				patch:    JSONObject{},
			})
			partition.sources = next
			partitions[*gvk] = partition
		} else {
			partitions[*gvk] = PatchPartition{
				gvk:  *gvk,
				base: JSONObject{},
				sources: []PatchSource{{
					name:     name,
					original: resource,
					patch:    JSONObject{},
				}},
			}
		}
	}
	outputPartitions := map[schema.GroupVersionKind]PatchPartition{}
	for gvk, partition := range partitions {
		patchMeta, err := pg.schemaClient.GetPatchMetadata(gvk)
		if err != nil {
			return nil, err
		}
		partition.base = cloneJSON(partition.sources[0].original)
		for i := 1; i < len(partition.sources); i++ {
			other := partition.sources[i]
			patch, err := calculatePatch(other.original, partition.base, patchMeta)
			if err != nil {
				return nil, err
			}
			nextBase, err := subtractObject(partition.base, patch, k8spatch.PatchMeta{}, patchMeta)
			if err != nil {
				return nil, err
			}
			partition.base = nextBase
		}
		for i := 0; i < len(partition.sources); i++ {
			item := partition.sources[i]
			patch, err := calculatePatch(partition.base, item.original, patchMeta)
			if err != nil {
				return nil, err
			}
			orderedPatch, err := ExecutePatchOrdering(patch)
			if err != nil {
				return nil, err
			}
			item.patch = orderedPatch
			partition.sources[i] = item
		}
		outputPartitions[gvk] = partition
	}

	results := make([]PatchPartition, 0, len(outputPartitions))
	for _, value := range outputPartitions {
		results = append(results, value)
	}
	return results, nil
}

func GetResourceName(resource JSONObject) (string, error) {
	if metadata, ok := resource["metadata"]; ok {
		if typedMetadata, ok := metadata.(JSONObject); ok {
			if name, ok := typedMetadata["name"]; ok {
				if typedName, ok := name.(string); ok {
					return typedName, nil
				}
			}
		}
	}
	return "", fmt.Errorf("required attribute 'name' not found in resource metadata")
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
		if bValue, ok := b[k]; ok {
			switch typedAValue := aValue.(type) {
			case JSONObject:
				if typedBValue, ok := bValue.(JSONObject); ok {
					lookupMeta, patchMeta, err := patchContext.LookupPatchMetadataForStruct(k)
					if err != nil {
						return nil, err
					}
					result[k], err = subtractObject(typedAValue, typedBValue, patchMeta, lookupMeta)
					if err != nil {
						return nil, err
					}
					continue
				} else {
					log.Default().Printf("unexpected type mismatch while evaluating key '%s'\n", k)
				}
			case JSONArray:
				if typedBValue, ok := bValue.(JSONArray); ok {
					lookupMeta, patchMeta, err := patchContext.LookupPatchMetadataForSlice(k)
					if err != nil {
						return nil, err
					}
					if shouldSubtractList(patchMeta) {
						result[k], err = subtractList(typedAValue, typedBValue, patchMeta, lookupMeta)
						if err != nil {
							return nil, err
						}
						continue
					}
				} else {
					log.Default().Printf("unexpected type mismatch while evaluating key '%s'\n", k)
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
