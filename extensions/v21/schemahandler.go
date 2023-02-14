package v21

import (
	"fmt"
	"io"

	"github.com/Perachi0405/ownediparse/errs"
	"github.com/Perachi0405/ownediparse/extensions/v21/fileformat"
	"github.com/Perachi0405/ownediparse/extensions/v21/fileformat/edi"
	"github.com/Perachi0405/ownediparse/extensions/v21/transform"
	v21validation "github.com/Perachi0405/ownediparse/extensions/v21/validation"
	"github.com/Perachi0405/ownediparse/schemahandler"
	"github.com/Perachi0405/ownediparse/transformctx"
	"github.com/Perachi0405/ownediparse/validation"
)

const (
	version = "omni.2.1"
)

// CreateParams allows user of this 'omni.2.1' schema handler to provide creation customization.
type CreateParams struct {
	CustomFileFormats []fileformat.FileFormat
	// Deprecated.
	CustomParseFuncs transform.CustomParseFuncs
}

// CreateSchemaHandler parses, validates and creates an omni-schema based handler.
func CreateSchemaHandler(ctx *schemahandler.CreateCtx) (schemahandler.SchemaHandler, error) {
	// fmt.Println("createschemahandler ctx", string(ctx.Content))  // json inputted schema - json structured
	fmt.Println("createschemahandler ctx name", ctx.Name)        //test-schema
	fmt.Println("createschemahandler ctx func", ctx.CustomFuncs) //the different functions in the schemaHandler in Omniv21
	fmt.Println("schema handler Executed omniv21")
	fmt.Println("Version", ctx.Header.ParserSettings.Version)
	if ctx.Header.ParserSettings.Version != version {
		return nil, errs.ErrSchemaNotSupported
	}
	// First do a `transform_declarations` json schema validation
	err := validation.SchemaValidate(ctx.Name, ctx.Content, v21validation.JSONSchemaTransformDeclarations) //validating the TransformDeclaration json schema
	fmt.Println("schemahandler.go omniv21", err)                                                           //nil
	if err != nil {
		// err is already context formatted.
		return nil, err
	}
	finalOutputDecl, err := transform.ValidateTransformDeclarations(
		ctx.Content, ctx.CustomFuncs, customParseFuncs(ctx))
	fmt.Println("schemahandler finaloutputdecl", finalOutputDecl) //Transformed array contains the data type unknown 0xc00055c3c0
	if err != nil {
		return nil, fmt.Errorf(
			"schema '%s' 'transform_declarations' validation failed: %s",
			ctx.Name, err.Error())
	}
	for _, fileFormat := range fileFormats(ctx) {
		formatRuntime, err := fileFormat.ValidateSchema(
			ctx.Header.ParserSettings.FileFormatType,
			ctx.Content,
			finalOutputDecl)
		if err == errs.ErrSchemaNotSupported {
			continue
		}
		if err != nil {
			// error from FileFormat is already context formatted.
			return nil, err
		}
		return &schemaHandler{
			ctx:             ctx,
			fileFormat:      fileFormat,
			formatRuntime:   formatRuntime,
			finalOutputDecl: finalOutputDecl,
		}, nil
	}
	return nil, errs.ErrSchemaNotSupported
}

func customParseFuncs(ctx *schemahandler.CreateCtx) transform.CustomParseFuncs {
	fmt.Println("customParseFuncs..")
	if ctx.CreateParams == nil {
		return nil
	}
	params, ok := ctx.CreateParams.(*CreateParams)
	fmt.Println("Params", params)
	if !ok {
		return nil
	}
	if len(params.CustomParseFuncs) == 0 {
		return nil
	}
	return params.CustomParseFuncs
}

//return the file format
func fileFormats(ctx *schemahandler.CreateCtx) []fileformat.FileFormat {
	fmt.Println("fileformats..")
	fmt.Println("fileformats.. ctx.Name", ctx.Name)
	//fmt.Println("fileformats.. ctx.content", string(ctx.Content)) //the whole JSON Schema
	formats := []fileformat.FileFormat{

		edi.NewEDIFileFormat(ctx.Name),
	}
	fmt.Println("fileFormats schemahandler", formats) // array of file formats
	if ctx.CreateParams == nil {
		return formats
	}
	params, ok := ctx.CreateParams.(*CreateParams)
	if !ok {
		return formats
	}
	// If caller specifies a list of custom FileFormats, we'll give them priority
	// over builtin ones.
	return append(params.CustomFileFormats, formats...)
}

type schemaHandler struct {
	ctx             *schemahandler.CreateCtx
	fileFormat      fileformat.FileFormat
	formatRuntime   interface{}
	finalOutputDecl *transform.Decl
}

func (h *schemaHandler) NewIngester(ctx *transformctx.Ctx, input io.Reader) (schemahandler.Ingester, error) {
	fmt.Println("newIngester ctx", ctx) //&{test-input map[] <nil> <nil>}
	//fmt.Println("input newIngester", input)                                               //input file in byte type data.
	//fmt.Println("newIngester ctx", ctx.InputName)                                         //test-input
	//fmt.Println("fileFormat", h.fileFormat)                                               //{test-schema}
	//fmt.Println("FormatRuntime", h.formatRuntime)                                         //&{0xc000dac5f0 }
	reader, err := h.fileFormat.CreateFormatReader(ctx.InputName, input, h.formatRuntime) //here test-input , input file, &{0xc000dac5f0 } are the inputs
	fmt.Println("newIngester", reader)                                                    //&{test-input {<nil> []} 0xc0009140f0 [{0xc00098a230 0xc000120780 0 0} {0xc000152ee0 <nil> 0 0}] <nil> <nil> {false  [] []}}
	if err != nil {
		return nil, err
	}
	return &ingester{
		finalOutputDecl:  h.finalOutputDecl,
		customFuncs:      h.ctx.CustomFuncs,
		customParseFuncs: customParseFuncs(h.ctx),
		ctx:              ctx,
		reader:           reader,
	}, nil
}
