package v21

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Perachi0405/ownediparse/customfuncs"
	"github.com/Perachi0405/ownediparse/errs"
	"github.com/Perachi0405/ownediparse/extensions/v21/fileformat"
	"github.com/Perachi0405/ownediparse/extensions/v21/transform"
	"github.com/Perachi0405/ownediparse/idr"
	"github.com/Perachi0405/ownediparse/schemahandler"
	"github.com/Perachi0405/ownediparse/transformctx"
)

type rawRecord struct {
	node *idr.Node
}

func (rr *rawRecord) Raw() interface{} {
	return rr.node
}

// Checksum returns a stable MD5(v3) hash of the rawRecord.
func (rr *rawRecord) Checksum() string {
	hash, _ := customfuncs.UUIDv3(nil, idr.JSONify2(rr.node))
	return hash
}

type ingester struct {
	finalOutputDecl  *transform.Decl
	customFuncs      customfuncs.CustomFuncs
	customParseFuncs transform.CustomParseFuncs // Deprecated.
	ctx              *transformctx.Ctx
	reader           fileformat.FormatReader
	rawRecord        rawRecord
}

// Read ingests a raw record from the input stream, transforms it according the given schema and return
// the raw record, transformed JSON bytes.
func (g *ingester) Read() (schemahandler.RawRecord, []byte, error) {
	fmt.Println("ingester", g)
	if g.rawRecord.node != nil {
		g.reader.Release(g.rawRecord.node)
		g.rawRecord.node = nil
	}
	fmt.Println("Before Read()")
	n, err := g.reader.Read()
	fmt.Println("Read() ingester", n) //nil
	if n != nil {
		g.rawRecord.node = n
		fmt.Println("g.rawRecord.node", g.rawRecord.node)
	}
	if err != nil {
		// Read() supposed to have already done CtxAwareErr error wrapping. So directly return.
		return nil, nil, err
	}
	fmt.Println("ingester before result g.ctx", g.ctx)
	fmt.Println("ingester before result g.customfuncs", g.customFuncs)
	fmt.Println("ingester before result g.customParseFuncs", g.customParseFuncs)
	result, err := transform.NewParseCtx(g.ctx, g.customFuncs, g.customParseFuncs).ParseNode(n, g.finalOutputDecl) //here the parsing takes place.
	//fmt.Println("Result transform.NewParseCtx", result)                                                            //maped version of the segments map[AMT:map[AmountQualifierCode:TT MonetaryAmount:15922.26] type
	if err != nil {
		// ParseNode() error not CtxAwareErr wrapped, so wrap it.
		// Note errs.ErrorTransformFailed is a continuable error.
		return nil, nil, errs.ErrTransformFailed(g.fmtErrStr("fail to transform. err: %s", err.Error()))
	}
	transformed, err := json.Marshal(result)
	//fmt.Println("transformed Read()", transformed) //transformed JSON data
	fmt.Println("rawrecord ingester", &g.rawRecord) //nil
	return &g.rawRecord, transformed, err
}

func (g *ingester) IsContinuableError(err error) bool {
	return errs.IsErrTransformFailed(err) || g.reader.IsContinuableError(err)
}

func (g *ingester) FmtErr(format string, args ...interface{}) error {
	return errors.New(g.fmtErrStr(format, args...))
}

func (g *ingester) fmtErrStr(format string, args ...interface{}) string {
	return g.reader.FmtErr(format, args...).Error()
}
