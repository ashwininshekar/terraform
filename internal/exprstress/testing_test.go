package exprstress

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestTestCase(t *testing.T) {
	// Since TestCase directly generates errors in its given testing.T we
	// can't really test failing cases here, but we can at least use this
	// to test some successful cases to show that the expression evaluation
	// is happening as expected.
	tests := []struct {
		Expr     string
		Expected Expected
	}{
		{
			`1`,
			Expected{
				Type: cty.Number,
				Mode: SpecifiedValue,
			},
		},
		{
			`true`,
			Expected{
				Type: cty.Bool,
				Mode: SpecifiedValue,
			},
		},
		{
			`1 + 1`,
			Expected{
				Type: cty.Number,
				Mode: SpecifiedValue,
			},
		},
		{
			`null`,
			Expected{
				Type: cty.DynamicPseudoType,
				Mode: NullValue,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Expr, func(t *testing.T) {
			t.Logf("expression: %s", test.Expr)
			TestCase(t, test.Expr, test.Expected)
		})
	}
}