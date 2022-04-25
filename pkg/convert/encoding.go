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

func ParseYAMLFileIntoJSONObjects(y []byte) ([]JSONObject, error) {
	result := make([]JSONObject, 0)
	reader := bytes.NewReader(y)
	decoder := yaml.NewDecoder(reader)
	for {
		var yamlObj map[string]interface{}
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
		var jsonObj any
		err = json.Unmarshal(jsonBytes, &jsonObj)
		if err != nil {
			return nil, err
		}
		switch typedJSONObj := jsonObj.(type) {
		case JSONObject:
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

func cloneJSON(o JSONObject) JSONObject {
	var cloned JSONObject
	sourceBytes, _ := json.Marshal(o)
	_ = json.Unmarshal(sourceBytes, &cloned)
	return cloned
}
