package ownediparse

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	omniv21 "github.com/Perachi0405/ownediparse/extensions/v21"
	v21 "github.com/Perachi0405/ownediparse/extensions/v21/customfuncs"

	"github.com/jf-tech/go-corelib/ios"

	"github.com/Perachi0405/ownediparse/customfuncs"
	"github.com/Perachi0405/ownediparse/errs"
	"github.com/Perachi0405/ownediparse/header"
	"github.com/Perachi0405/ownediparse/schemahandler"
	"github.com/Perachi0405/ownediparse/transformctx"
	"github.com/Perachi0405/ownediparse/validation"
)

// Schema is an interface that represents a schema used by omniparser.
// One instance of Schema is associated with one and only one schema.
// The instance of Schema can be reused for ingesting and transforming
// multiple input files/streams, as long as they are all intended for the
// same schema.
// Each ingestion/transform, however, needs a separate instance of
// Transform. A Transform must not be shared and reused across different
// input files/streams.
// While the same instance of Schema can be shared across multiple threads,
// Transform is not multi-thread safe. All operations on it must be done
// within the same go routine.
type Schema interface {
	NewTransform(name string, input io.Reader, ctx *transformctx.Ctx) (Transform, error)
	Header() header.Header
	Content() []byte
}

//structure of schema
type schema struct {
	name    string
	header  header.Header
	content []byte
	handler schemahandler.SchemaHandler
}

// Extension allows user of omniparser to add new schema handlers, and/or new custom functions
// in addition to the builtin handlers and functions.
type Extension struct {
	CreateSchemaHandler       schemahandler.CreateFunc //checks the given schema is supported by its associated schema handler
	CreateSchemaHandlerParams interface{}
	CustomFuncs               customfuncs.CustomFuncs //omniparse custom functions
}

// 'omni.2.1' extension
var (
	defaultExt = Extension{
		CreateSchemaHandler: omniv21.CreateSchemaHandler, //creates a schema handler
		CustomFuncs:         customfuncs.Merge(customfuncs.CommonCustomFuncs, v21.OmniV21CustomFuncs),
	}
)

// NewSchema creates a new instance of Schema. Caller can use the optional Extensions for customization.
// NewSchema will scan through exts left to right to find the first extension with a schema handler (specified
// by CreateSchemaHandler field) that supports the input schema. If no ext provided or no ext with a handler
// that supports the schema, then NewSchema will fall back to builtin extension (currently for schema version
// 'omni.2.1'). If the input schema is still not supported by builtin extension, NewSchema will fail with
// ErrSchemaNotSupported. Each extension much be fully self-contained meaning all the custom functions it
// intends to use in the schemas supported by it must be included in the same extension.
func NewSchema(name string, schemaReader io.Reader, exts ...Extension) (Schema, error) { //
	fmt.Println("Invoked the Newschema")
	fmt.Println("Inside schemaReader", schemaReader)
	content, err := ioutil.ReadAll(schemaReader) //selected schema
	// fmt.Println("NewSchema schema.go", string(content)) //reads the selected schema and try to
	if err != nil {
		return nil, fmt.Errorf("unable to read schema '%s': %s", name, err.Error())
	}
	// validate the universal parser_settings header schema.
	err = validation.SchemaValidate(name, content, validation.JSONSchemaParserSettings)
	// fmt.Println("Error Newchema()", err)
	if err != nil {
		// The err from validation.SchemaValidate is already context formatted.
		return nil, err
	}
	var h header.Header
	// fmt.Println("Header Details", h)                           //getting Nil
	// fmt.Println("JSON Unmarshal", json.Unmarshal(content, &h)) //getting Nil
	// parser_settings has just been json schema validated. so unmarshaling will not go wrong.
	_ = json.Unmarshal(content, &h)

	allExts := append([]Extension(nil), exts...)
	fmt.Println("Data allExts1", allExts) //getting []
	allExts = append(allExts, defaultExt)
	fmt.Println("allExts", allExts)

	fmt.Println("Default", defaultExt)
	// fmt.Println("Data allExts 0", allExts[0].CreateSchemaHandler)
	// fmt.Println("Data allExts 1", allExts[0].CreateSchemaHandlerParams)
	// fmt.Println("Data allExts 2", allExts[0].CustomFuncs) // customfunctions
	// marsh, err := json.Marshal(allExts[0].CreateSchemaHandler)
	// fmt.Println("Marshaling", marsh)
	// str1 := fmt.Sprintf("%s", marsh)
	// fmt.Println("String convertion", str1)

	// loop through the extensions
	for _, ext := range allExts {
		fmt.Println("for loop ext", ext)
		fmt.Println("Index 0", allExts[0])
		if ext.CreateSchemaHandler == nil {
			continue
		}
		handler, err := ext.CreateSchemaHandler(&schemahandler.CreateCtx{ //using each extension we create a schemahandler
			Name:         name,
			Header:       h,
			Content:      content,
			CustomFuncs:  ext.CustomFuncs,               //interface{}
			CreateParams: ext.CreateSchemaHandlerParams, //interface{}
		})
		// fmt.Println("Header schema.go", handler)
		if err == errs.ErrSchemaNotSupported {
			continue
		}
		if err != nil {
			// The err from handler's CreateSchemaHandler is already ctxAwareErr formatted, so directly return.
			return nil, err
		}
		// check := Schema

		//save the values in the schema structure
		return &schema{
			name:    name,
			header:  h,
			content: content,
			handler: handler, //save the handler in the handler place
		}, nil
	}
	return nil, errs.ErrSchemaNotSupported
}

// NewTransform creates and returns an instance of Transform for a given input stream.
//receiver function
func (s *schema) NewTransform(name string, input io.Reader, ctx *transformctx.Ctx) (Transform, error) { //type what
	fmt.Println("handler", s.handler)
	fmt.Println("header", s.header)
	fmt.Println("Sample input", input)
	// fmt.Println("inputReader", input) //the EDI input file
	//fmt.Println("encoding", s.header.ParserSettings.WrapEncoding(input)) //EDI input file ending with 0 -1
	br, err := ios.StripBOM(s.header.ParserSettings.WrapEncoding(input))
	//fmt.Println("NewTransform br", br) //byte typed data with atlast special value is appended.
	//fmt.Println("NewTransform err", err)//nil
	if err != nil {
		return nil, err
	}
	if ctx.InputName != name {
		ctx.InputName = name
	}
	fmt.Println("before NewIngester invoke")
	ingester, err := s.handler.NewIngester(ctx, br)
	fmt.Println("ingester schema.go", ingester) //custom functions
	if err != nil {
		return nil, err
	}
	// If caller already specified a way to do context aware error formatting, use it;
	// otherwise (vast majority cases), use the Ingester (which implements CtxAwareErr
	// interface) created by the schema handler.
	if ctx.CtxAwareErr == nil {
		ctx.CtxAwareErr = ingester
	}
	return &transform{ingester: ingester}, nil
}

// Header returns the schema header.
func (s *schema) Header() header.Header {
	fmt.Println("Header", s.header) //unknown data
	return s.header
}

// Content returns the schema content.
func (s *schema) Content() []byte {
	//fmt.Println("Return Data", s.content) no logs found
	return s.content
}
