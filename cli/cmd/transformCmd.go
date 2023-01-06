package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jf-tech/go-corelib/ios"
	"github.com/jf-tech/go-corelib/jsons"
	"github.com/jf-tech/go-corelib/strs"
	"github.com/spf13/cobra"

	"github.com/jf-tech/omniparser"
	"github.com/jf-tech/omniparser/transformctx"
)

var (
	transformCmd = &cobra.Command{
		Use:   "transform",
		Short: "Transforms input to desired output based on a schema.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := doTransform(); err != nil { //invoke the doTransform()
				fmt.Println() // to sure cobra cli always write out "Error: ..." on a new line.
				return err
			}
			return nil
		},
	}
	schema string
	input  string
)

func init() {
	fmt.Println("Init in transformCmd")
	transformCmd.Flags().StringVarP(&schema, "schema", "s", "", "schema file (required)")
	_ = transformCmd.MarkFlagRequired("schema")

	transformCmd.Flags().StringVarP(
		&input, "input", "i", "", "input file (optional; if not specified, stdin/pipe is used)")
}

func openFile(label string, filepath string) (io.ReadCloser, error) {
	fmt.Println("Openfile in transformCmd.go")
	if !ios.FileExists(schema) {
		return nil, fmt.Errorf("%s file '%s' does not exist", label, filepath)
	}
	return os.Open(filepath)
}

func doTransform() error {
	fmt.Println("doTransfrom() transformCmd")
	schemaName := filepath.Base(schema)
	fmt.Println("schemName", schemaName)
	schemaReadCloser, err := openFile("schema", schema)
	fmt.Println("schemaReadCloser", schemaReadCloser)
	if err != nil {
		return err
	}
	defer schemaReadCloser.Close()

	inputReadCloser := io.ReadCloser(nil)
	inputName := ""
	if strs.IsStrNonBlank(input) { //checking the input [transformCmd] structure is blank or not
		inputName = filepath.Base(input)
		fmt.Println("InputName doTransform()", inputName)
		inputReadCloser, err = openFile("input", input)
		fmt.Println("inputReadCloser doTransform()", inputReadCloser)
		if err != nil {
			return err
		}
		defer inputReadCloser.Close()
	} else {
		inputName = "(stdin)"
		inputReadCloser = os.Stdin
		// Note we don't defer Close() on this since os/golang runtime owns it.
	}

	schema, err := omniparser.NewSchema(schemaName, schemaReadCloser) //creates a new instance of the Schema
	fmt.Println("schema doTransform()", schema)
	if err != nil {
		return err
	}

	transform, err := schema.NewTransform(inputName, inputReadCloser, &transformctx.Ctx{})
	fmt.Println("transform doTransform()", transform)
	if err != nil {
		return err
	}

	doOne := func() (string, error) {
		b, err := transform.Read()
		if err != nil {
			return "", err
		}
		return strings.Join(
			strs.NoErrMapSlice(
				strings.Split(jsons.BPJ(string(b)), "\n"),
				func(s string) string { return "\t" + s }),
			"\n"), nil
	}

	record, err := doOne()
	if err == io.EOF {
		fmt.Println("[]")
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Printf("[\n%s", record)
	for {
		record, err = doOne()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fmt.Printf(",\n%s", record)
	}
	fmt.Println("\n]")
	return nil
}
