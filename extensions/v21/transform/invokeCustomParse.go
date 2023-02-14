package transform

import (
	"fmt"
	"github/Perachi0405/ownediparse/idr"
	"github/Perachi0405/ownediparse/transformctx"
	"reflect"
)

// CustomParseFuncType represents a custom_parse function type. Deprecated. Use customfuncs.CustomFuncType.
type CustomParseFuncType func(*transformctx.Ctx, *idr.Node) (interface{}, error)

// CustomParseFuncs is a map from custom_parse names to an actual custom parse functions. Deprecated. Use
// customfuncs.CustomFuncs.
type CustomParseFuncs = map[string]CustomParseFuncType

//log not found
func (p *parseCtx) invokeCustomParse(customParseFn CustomParseFuncType, n *idr.Node) (interface{}, error) {
	fmt.Println("invokeCustomParse...")
	result := reflect.ValueOf(customParseFn).Call(
		[]reflect.Value{
			reflect.ValueOf(p.transformCtx),
			reflect.ValueOf(n),
		})
	fmt.Println("result invokeCustomParse", result)
	// result[0] - result
	// result[1] - error
	if result[1].Interface() == nil {
		return result[0].Interface(), nil
	}
	fmt.Println("return result[1]", result[1])
	return nil, result[1].Interface().(error)
}
