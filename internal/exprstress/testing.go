package exprstress

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/lang"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

// TestExpression tries to evaluate the given test expression and returns
// errors if parsing or evaluation fail or if a successful result doesn't
// conform to the expression's expectations.
func TestExpression(expr Expression) []error {
	src := ExpressionSourceBytes(expr)
	expected := expr.ExpectedResult()
	return testExpression(src, expected)
}

// TestCase is a helper function to help with permanently capturing interesting
// test cases and their expected results as examples in hand-written unit tests.
//
// The "tfexprstress run" command will generate example calls to this function
// for each failure it encounters, to help with setting up a reproducible
// test case for further debugging.
//
// TestCase generates errors on the given testing.T if the test fails, but it
// doesn't prevent further execution of subsequent test code. In case the
// outcome of the test case needs to conditionally block further test
// execution, TestCase returns true if it detected at least one error. Callers
// may ignore the return value if the test outcome is immeterial for any
// subsequent test code.
func TestCase(t *testing.T, exprSrc string, expected Expected) bool {
	t.Helper()
	errs := testExpression([]byte(exprSrc), expected)
	for _, err := range errs {
		t.Error(err)
	}
	return len(errs) > 0
}

func testExpression(src []byte, expected Expected) (errs []error) {
	defer func() {
		// Since expression evaluation is typically self-contained we'll
		// try to present panics as normal errors so that we can potentially
		// print out a useful reproduction case message and keep testing.
		if r := recover(); r != nil {
			errs = append(errs, errorForPanic(r))
		}
	}()

	expr, hclDiags := hclsyntax.ParseExpression(src, "", hcl.InitialPos)
	for _, diag := range hclDiags {
		errs = append(errs, diag)
	}
	if len(errs) > 0 {
		// If parsing failed then we won't even try evaluation
		return errs
	}

	scope := &lang.Scope{
		// TODO: Fill this out properly
	}

	v, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)
	for _, diag := range diags {
		desc := diag.Description()
		var rng hcl.Range
		if subject := diag.Source().Subject; subject != nil {
			rng = subject.ToHCL()
		}
		errs = append(errs, fmt.Errorf("[%s] %s: %s", rng, desc.Summary, desc.Detail))
	}
	if len(errs) > 0 {
		// If evaluation failed then we won't check against the expected value
		return errs
	}

	if v == cty.NilVal {
		// NilVal is never a valid result for a successful evaluation
		errs = append(errs, fmt.Errorf("result is cty.NilVal"))
		return errs
	}

	if got, want := v.Type(), expected.Type; !want.Equals(got) {
		errs = append(errs, fmt.Errorf(
			"wrong result type\ngot:  %swant: %s",
			ctydebug.TypeString(got),
			ctydebug.TypeString(want),
		))
	}

	var gotMode ValueMode
	switch {
	case v.IsNull():
		gotMode = NullValue
	case !v.IsKnown():
		gotMode = UnknownValue
	default:
		gotMode = SpecifiedValue
	}
	if gotMode != expected.Mode {
		errs = append(errs, fmt.Errorf(
			"result has wrong mode\ngot:  %s\nwant: %s",
			gotMode, expected.Mode,
		))
	}

	if got, want := v.IsMarked(), expected.Sensitive; got != want {
		errs = append(errs, fmt.Errorf(
			"wrong result sensitivity\ngot:  %#v\nwant: %#v",
			got, want,
		))
	}

	return errs
}

type panicError struct {
	Value interface{}
	Stack []byte
}

func errorForPanic(val interface{}) error {
	return panicError{
		Value: val,
		Stack: debug.Stack(),
	}
}

func (e panicError) Error() string {
	return fmt.Sprintf("panic during expression evaluation: %s\n%s", e.Value, e.Stack)
}