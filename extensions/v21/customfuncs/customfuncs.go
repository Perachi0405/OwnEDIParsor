package customfuncs

import (
	"fmt"

	"github.com/Perachi0405/ownediparse/customfuncs"
	"github.com/Perachi0405/ownediparse/idr"
	"github.com/Perachi0405/ownediparse/transformctx"
)

// OmniV21CustomFuncs contains 'omni.2.1' specific custom funcs.
var OmniV21CustomFuncs = map[string]customfuncs.CustomFuncType{ //customfuncs.CustomFuncType specify an interface
	// keep these custom funcs lexically sorted
	"copy":                    CopyFunc,              //Log not found
	"javascript":              JavaScript,            //Log found for resetcache()
	"javascript_with_context": JavaScriptWithContext, //Log not found
}

// CopyFunc copies the current contextual idr.Node and returns it as a JSON marshaling friendly interface{}.
func CopyFunc(_ *transformctx.Ctx, n *idr.Node) (interface{}, error) {
	fmt.Println("Inside the copyFunc")
	return idr.J2NodeToInterface(n, true), nil
}
