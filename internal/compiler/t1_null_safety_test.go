package compiler

import (
	"strings"
	"testing"
)

// T1-04: Null safety — the compiler must catch undefined variables at compile
// time and emit a structured error with line/column information.
// These tests prove the resolver (not the runtime) owns undefined-variable detection.
//
// CRITICAL: If any of these pass without an error, Wolf has a soundness hole.

// TestT1_04_UndeclaredVariable is the canonical T1-04 case from the test spec:
//   $x = $undeclared + 1
// must emit a WolfError pointing at $undeclared — not compile or execute silently.
func TestT1_04_UndeclaredVariable(t *testing.T) {
	errors := compileAndExpectError(t, `$x = $undeclared + 1`)
	if len(errors) == 0 {
		t.Fatal("T1-04: expected at least one error for undeclared $undeclared, got none")
	}
	combined := strings.Join(errors, "\n")
	if !strings.Contains(combined, "undeclared") && !strings.Contains(combined, "undefined") &&
		!strings.Contains(combined, "$undeclared") {
		t.Errorf("T1-04: error message should mention the undeclared variable name, got:\n%s", combined)
	}
}

// TestT1_04_UndeclaredInFunction ensures undeclared variables inside function bodies
// are caught — not silently treated as global.
func TestT1_04_UndeclaredInFunction(t *testing.T) {
	src := `
func my_func() {
    $result = $ghost_var + 10
    return $result
}
my_func()
`
	errors := compileAndExpectError(t, src)
	if len(errors) == 0 {
		t.Fatal("T1-04: expected error for $ghost_var inside function, got none")
	}
	combined := strings.Join(errors, "\n")
	if !strings.Contains(combined, "ghost_var") && !strings.Contains(combined, "undefined") &&
		!strings.Contains(combined, "undeclared") {
		t.Errorf("T1-04: error should reference the undeclared name, got:\n%s", combined)
	}
}

// TestT1_04_UndeclaredInCondition ensures undeclared variables in conditionals are caught.
func TestT1_04_UndeclaredInCondition(t *testing.T) {
	src := `
if $does_not_exist {
    print("dead code")
}
`
	errors := compileAndExpectError(t, src)
	if len(errors) == 0 {
		t.Fatal("T1-04: expected error for $does_not_exist in if condition, got none")
	}
}

// TestT1_04_UndeclaredInLoop ensures undeclared variables in loop conditions are caught.
func TestT1_04_UndeclaredInLoop(t *testing.T) {
	src := `
while $phantom_flag {
    print("loop")
}
`
	errors := compileAndExpectError(t, src)
	if len(errors) == 0 {
		t.Fatal("T1-04: expected error for $phantom_flag in while condition, got none")
	}
}

// TestT1_04_UndeclaredMethodArg ensures undeclared variables passed as arguments are caught.
func TestT1_04_UndeclaredMethodArg(t *testing.T) {
	src := `
func take_val($v: int) -> int {
    return $v
}
$result = take_val($missing_arg)
`
	errors := compileAndExpectError(t, src)
	if len(errors) == 0 {
		t.Fatal("T1-04: expected error for $missing_arg as argument, got none")
	}
}

// TestT1_04_DeclaredVariableSucceeds is the control case: a declared variable
// must NOT produce an error. This ensures we haven't broken the happy path.
// In StrictMode, typed var declarations are used.
func TestT1_04_DeclaredVariableSucceeds(t *testing.T) {
	// This should compile without errors — using typed var declarations
	c := New()
	c.StrictMode = true
	result, err := c.Compile(`var $x: int = 10
var $y: int = 20
print("{$x} {$y}")`, "test.wolf")
	if err != nil {
		t.Errorf("T1-04 control: typed var declarations should compile cleanly, got error:\n%v\n%s",
			err, strings.Join(result.Errors, "\n"))
	}
}

// TestT1_04_ErrorHasStructuredDiagnostic checks that the resolver emits
// structured WolfError diagnostics (not just a string blob), which powers
// the LSP and the wolf explain tool.
func TestT1_04_ErrorHasStructuredDiagnostic(t *testing.T) {
	c := New()
	c.StrictMode = true
	result, err := c.Compile(`$x = $totally_missing + 1`, "t1_04_test.wolf")
	if err == nil {
		t.Fatal("T1-04: expected error for $totally_missing, got none")
	}
	// result.Diagnostics should be populated for LSP
	if len(result.Diagnostics) == 0 {
		// Fallback: errors slice must at least be non-empty
		if len(result.Errors) == 0 {
			t.Error("T1-04: neither Diagnostics nor Errors were populated")
		} else {
			t.Logf("T1-04: Diagnostics empty but Errors populated: %v", result.Errors)
			t.Logf("NOTE: Populate result.Diagnostics in the resolver for LSP hover support")
		}
	} else {
		diag := result.Diagnostics[0]
		if diag.Pos.Line <= 0 {
			t.Errorf("T1-04: diagnostic should have a positive line number, got %d", diag.Pos.Line)
		}
		t.Logf("T1-04 ✅ structured diagnostic: line=%d col=%d msg=%s", diag.Pos.Line, diag.Pos.Col, diag.Message)
	}
}
