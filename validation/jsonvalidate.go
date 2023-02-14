package validation

//go:generate sh -c "go run gen/gen.go -json parserSettings.json -varname JSONSchemaParserSettings > ./parserSettings.go"

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// SchemaValidate validates a schema based on its JSON schema. Any validation error, if
// present, is context formatted, i.e. schema name is prefixed in the error msg.
//Validating the transform_declaration
func SchemaValidate(schemaName string, schemaContent []byte, jsonSchema string) error {
	fmt.Println("validation jsonvalidate Schemavalidate")
	// fmt.Println("schemacontent", string(schemaContent)) //Normal json schema file[input]
	// fmt.Println("jsonschema", jsonSchema)               // reads the json schema[configured]

	// fmt.Println("schema validate", jsonSchema) //normal common schema to validate the input json schema
	//fmt.Println("targetSchemaLoader", string(schemaContent)) //Inputed JSON schema

	jsonSchemaLoader := gojsonschema.NewStringLoader(jsonSchema) //contains the schema
	targetSchemaLoader := gojsonschema.NewBytesLoader(schemaContent)
	result, err := gojsonschema.Validate(jsonSchemaLoader, targetSchemaLoader)
	fmt.Println("Result schemaValidate", result)
	if err != nil {
		return fmt.Errorf("unable to perform schema validation: %s", err)
	}
	if result.Valid() {
		return nil
	}
	var errs []string
	for _, err := range result.Errors() {
		errs = append(errs, err.String())
	}
	sort.Strings(errs)
	if len(errs) == 1 {
		return fmt.Errorf("schema '%s' validation failed: %s", schemaName, errs[0])
	}
	return fmt.Errorf("schema '%s' validation failed:\n%s", schemaName, strings.Join(errs, "\n"))
}
