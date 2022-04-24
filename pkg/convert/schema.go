package convert

import (
	"fmt"
	openapi_v3 "github.com/google/gnostic/openapiv3"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8spatch "k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/kube-openapi/pkg/util/proto"
	"os"
)

type SchemaClient struct {
	models proto.Models
}

func NewSchemaClient(schemaPath string) (*SchemaClient, error) {
	schemaData, err := os.ReadFile(schemaPath)
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
	sc := &SchemaClient{
		models,
	}
	return sc, nil
}
func (sc *SchemaClient) GetPatchMetadata(schemaName string) k8spatch.LookupPatchMeta {
	model := sc.models.LookupModel(schemaName)
	patchMeta := k8spatch.NewPatchMetaFromOpenAPI(model)
	return patchMeta
}
func (sc *SchemaClient) LookupResource(gvk schema.GroupVersionKind) proto.Schema {
	modelName := fmt.Sprintf("%s.%s.%s", gvk.Group, gvk.Version, gvk.Kind)
	return sc.models.LookupModel(modelName)
}
