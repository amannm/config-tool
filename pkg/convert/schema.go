package convert

import (
	"fmt"
	openapi_v3 "github.com/google/gnostic/openapiv3"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8spatch "k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/kube-openapi/pkg/util/proto"
	"os"
	"path"
	"strings"
)

type SchemaClient struct {
	schemaNameLookup map[string]*proto.Schema
	gvkLookup        map[schema.GroupVersionKind]*proto.Schema
}

func NewSchemaClient(schemaFolderPath string) (*SchemaClient, error) {
	entries, err := os.ReadDir(schemaFolderPath)
	if err != nil {
		return nil, err
	}
	schemas := map[string]*proto.Schema{}
	gvks := map[schema.GroupVersionKind]*proto.Schema{}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, "_openapi.json") {
			filePath := path.Join(schemaFolderPath, name)
			schemaData, err := os.ReadFile(filePath)
			if err != nil {
				return nil, err
			}
			doc, err := openapi_v3.ParseDocument(schemaData)
			if err != nil {
				return nil, err
			}
			models, err := proto.NewOpenAPIV3Data(doc)
			if err != nil {
				return nil, err
			}
			modelNames := models.ListModels()
			for _, modelName := range modelNames {
				modelSchema := models.LookupModel(modelName)
				_, ok := schemas[modelName]
				if !ok {
					schemas[modelName] = &modelSchema
				}
				modelGvks := parseGVKs(modelSchema)
				for _, modelGvk := range modelGvks {
					_, ok := gvks[modelGvk]
					if !ok {
						gvks[modelGvk] = &modelSchema
					}
				}
			}
		}
	}
	sc := &SchemaClient{
		schemas,
		gvks,
	}
	return sc, nil
}
func (sc *SchemaClient) GetPatchMetadata(gvk schema.GroupVersionKind) (k8spatch.LookupPatchMeta, error) {
	modelSchema, ok := sc.gvkLookup[gvk]
	if !ok {
		return nil, fmt.Errorf(fmt.Sprintf("resource schema not found for GVK: %s", gvk.String()))
	}
	patchMeta := k8spatch.NewPatchMetaFromOpenAPI(*modelSchema)
	return patchMeta, nil
}
func (sc *SchemaClient) GetSchemaByGVK(manifest JSONObject) (*proto.Schema, error) {
	gvk, err := ComputeGVK(manifest)
	if err != nil {
		return nil, err
	}
	modelSchema, ok := sc.gvkLookup[*gvk]
	if !ok {
		return nil, fmt.Errorf(fmt.Sprintf("resource schema not found for GVK: %s", gvk.String()))
	}
	return modelSchema, nil
}

const groupVersionKindExtensionKey = "x-kubernetes-group-version-kind"

func parseGVKs(s proto.Schema) []schema.GroupVersionKind {
	extensions := s.GetExtensions()
	gvkListResult := []schema.GroupVersionKind{}
	gvkExtension, ok := extensions[groupVersionKindExtensionKey]
	if !ok {
		return []schema.GroupVersionKind{}
	}
	gvkList, ok := gvkExtension.(JSONArray)
	if !ok {
		return []schema.GroupVersionKind{}
	}
	for _, gvk := range gvkList {
		gvkMap, ok := gvk.(JSONObject)
		if !ok {
			continue
		}
		group, ok := gvkMap["group"].(string)
		if !ok {
			continue
		}
		version, ok := gvkMap["version"].(string)
		if !ok {
			continue
		}
		kind, ok := gvkMap["kind"].(string)
		if !ok {
			continue
		}
		gvkListResult = append(gvkListResult, schema.GroupVersionKind{
			Group:   group,
			Version: version,
			Kind:    kind,
		})
	}
	return gvkListResult
}

func ComputeGVK(jsonObject JSONObject) (*schema.GroupVersionKind, error) {
	apiVersion, ok := jsonObject["apiVersion"]
	if !ok {
		return nil, fmt.Errorf("required property 'apiVersion' not found in resource declaration")
	}
	typedApiVersion, ok := apiVersion.(string)
	if !ok {
		return nil, fmt.Errorf("required property 'apiVersion' must be a string")
	}
	kind, ok := jsonObject["kind"]
	if !ok {
		return nil, fmt.Errorf("required property 'kind' not found in resource declaration")
	}
	typedKind, ok := kind.(string)
	if !ok {
		return nil, fmt.Errorf("required property 'kind' must be a string")
	}
	gvk := schema.FromAPIVersionAndKind(typedApiVersion, typedKind)
	return &gvk, nil
}
