package convert

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	k8syaml "sigs.k8s.io/yaml"
)

func JSONToYAML(j []byte) ([]byte, error) {
	return k8syaml.JSONToYAML(j)
}
func YAMLToJSON(y []byte) ([]byte, error) {
	return k8syaml.YAMLToJSONStrict(y)
}
func ParseYAMLFileIntoJSONObjects(y []byte) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0)
	reader := bytes.NewReader(y)
	decoder := yaml.NewDecoder(reader)
	for {
		var yamlObj interface{}
		err := decoder.Decode(&yamlObj)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if yamlObj == nil {
			break
		}
		yy, err := yaml.Marshal(yamlObj)
		if err != nil {
			return nil, err
		}
		jsonBytes, err := k8syaml.YAMLToJSONStrict(yy)
		if err != nil {
			return nil, err
		}
		var jsonObj interface{}
		err = json.Unmarshal(jsonBytes, &jsonObj)
		if err != nil {
			return nil, err
		}
		switch typedJSONObj := jsonObj.(type) {
		case map[string]interface{}:
			if len(typedJSONObj) == 0 {
				continue
			}
			result = append(result, typedJSONObj)
		default:
			return nil, fmt.Errorf("encountered unexpected non-object type")
		}
	}
	return result, nil
}

//
//func intersectAll(items []map[string]interface{}) (result map[string]interface{}) {
//	result = items[0]
//	for i := 1; i < len(items); i++ {
//		result = intersect(result, items[i])
//	}
//	patchMeta := k8spatch.NewPatchMetaFromOpenAPI(nil)
//	patch, err := k8spatch.CreateTwoWayMergePatch(nil, nil, patchMeta)
//	// calculate a patch of the diff from b to a
//	// negate the patch and apply to a
//	patched, err := k8spatch.StrategicMergePatchUsingLookupPatchMeta(nil, nil, patchMeta)
//	//
//	return result
//}
//
//func intersect(a map[string]interface{}, b map[string]interface{}) (result map[string]interface{}) {
//	result = map[string]interface{}{}
//	for k, v := range a {
//		switch aValue := v.(type) {
//		case map[string]interface{}:
//			switch bValue := b[k].(type) {
//			case map[string]interface{}:
//				result[k] = intersect(aValue, bValue)
//			}
//		case []interface{}:
//			switch bValue := b[k].(type) {
//			case []interface{}:
//				result[k] = intersect(aValue, bValue)
//			}
//		default:
//			if reflect.DeepEqual(aValue, b[k]) {
//				result[k] = aValue
//			}
//		}
//	}
//	return result
//}
