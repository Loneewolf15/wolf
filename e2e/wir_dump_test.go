package e2e_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestWIRDump(t *testing.T) {
	cmd := exec.Command("../wolf_out/main", "testdata/phase7_wir.wolf", "--dump-wir")
	
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	
	if err != nil {
		t.Fatalf("Native compiler failed: %v\nOutput: %s", err, outStr)
	}
	
	// Assert structural equivalence of WIR generated natively
	
	// 1. Should have the main function
	if !strings.Contains(outStr, "func main() -> i64") {
		t.Errorf("Expected WIR to contain 'func main() -> i64'")
	}
	
	// 2. Should have variable allocations
	if !strings.Contains(outStr, "%a = alloca ptr") {
		t.Errorf("Expected WIR to contain '%%a = alloca ptr'")
	}
	
	// 3. Should have binary op 'add' and 'slt' instruction
	if !strings.Contains(outStr, "add i64") {
		t.Errorf("Expected WIR to contain 'add i64' instruction")
	}
	if !strings.Contains(outStr, "slt") {
		t.Errorf("Expected WIR to contain 'slt' instruction for <")
	}
	
	// 4. Should have branch instructions
	if !strings.Contains(outStr, "cond_br") {
		t.Errorf("Expected WIR to contain 'cond_br' instruction")
	}
	
	t.Logf("WIR structural equivalence passed. Output excerpt:\n%s", outStr)
}

// TestWIREmitLLVM validates Phase 6 of the native pipeline (ADR-027):
// WIR → LLVM IR text format via --emit-llvm flag.
func TestWIREmitLLVM(t *testing.T) {
	cmd := exec.Command("../wolf_out/main", "testdata/phase7_wir.wolf", "--emit-llvm")

	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		t.Fatalf("Native compiler --emit-llvm failed: %v\nOutput: %s", err, outStr)
	}

	// 1. Must have LLVM module header
	if !strings.Contains(outStr, "ModuleID") {
		t.Errorf("Expected LLVM IR to contain 'ModuleID' header")
	}

	// 2. Must have function definition with correct signature
	if !strings.Contains(outStr, "define i64 @main()") {
		t.Errorf("Expected LLVM IR to contain 'define i64 @main()'")
	}

	// 3. Must have alloca instructions for local variables
	if !strings.Contains(outStr, "alloca ptr") {
		t.Errorf("Expected LLVM IR to contain 'alloca ptr' instructions")
	}

	// 4. Must have icmp (comparison) instructions — from WIR 'cmp' lowering
	if !strings.Contains(outStr, "icmp") {
		t.Errorf("Expected LLVM IR to contain 'icmp' comparison instruction")
	}

	// 5. Must have conditional branch — lowered from WIR cond_br
	if !strings.Contains(outStr, "br i1") {
		t.Errorf("Expected LLVM IR to contain 'br i1' conditional branch")
	}

	// 6. Must have unconditional branch for if.end / while.cond blocks
	if !strings.Contains(outStr, "br label") {
		t.Errorf("Expected LLVM IR to contain 'br label' unconditional branch")
	}

	// 7. Must return i64 from main
	if !strings.Contains(outStr, "ret i64 0") {
		t.Errorf("Expected LLVM IR to contain 'ret i64 0' return instruction")
	}

	// 8. Must have arithmetic: add and sub instructions
	if !strings.Contains(outStr, "add i64") {
		t.Errorf("Expected LLVM IR to contain 'add i64' arithmetic instruction")
	}
	if !strings.Contains(outStr, "sub i64") {
		t.Errorf("Expected LLVM IR to contain 'sub i64' arithmetic instruction")
	}

	t.Logf("LLVM IR emission passed (Phase 6 ADR-027). Output excerpt:\n%s", outStr[:min(len(outStr), 800)])
}

// TestWIRBothFlags verifies --dump-wir and --emit-llvm can be used together.
func TestWIRBothFlags(t *testing.T) {
	cmd := exec.Command("../wolf_out/main", "testdata/phase7_wir.wolf", "--dump-wir", "--emit-llvm")

	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		t.Fatalf("Native compiler with both flags failed: %v\nOutput: %s", err, outStr)
	}

	// Both WIR text and LLVM IR should be present
	if !strings.Contains(outStr, "func main() -> i64") {
		t.Errorf("Expected WIR output with both flags: missing 'func main() -> i64'")
	}
	if !strings.Contains(outStr, "define i64 @main()") {
		t.Errorf("Expected LLVM IR output with both flags: missing 'define i64 @main()'")
	}

	t.Logf("Both --dump-wir and --emit-llvm flags work together. OK.")
}

// min is a helper for safe string slicing in older Go toolchains.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
