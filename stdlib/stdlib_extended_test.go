package stdlib

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// ========== WolfStrings Tests ==========

var str WolfStrings

func TestStringsContains(t *testing.T) {
	if !str.Contains("Hello Wolf", "Wolf") {
		t.Error("Expected contains")
	}
	if str.Contains("Hello", "Wolf") {
		t.Error("Should not contain")
	}
}

func TestStringsStartsEndsWith(t *testing.T) {
	if !str.StartsWith("wolf-lang", "wolf") {
		t.Error("Expected startsWith")
	}
	if !str.EndsWith("wolf-lang", "lang") {
		t.Error("Expected endsWith")
	}
	if str.StartsWith("wolf", "WOLF") {
		t.Error("Case sensitive")
	}
}

func TestStringsSplitJoin(t *testing.T) {
	parts := str.Split("a,b,c", ",")
	if len(parts) != 3 {
		t.Fatalf("Expected 3 parts, got %d", len(parts))
	}
	joined := str.Join(parts, "-")
	if joined != "a-b-c" {
		t.Errorf("Expected 'a-b-c', got '%s'", joined)
	}
}

func TestStringsTrim(t *testing.T) {
	if str.Trim("  hello  ") != "hello" {
		t.Error("Trim failed")
	}
	if str.TrimChars("--hello--", "-") != "hello" {
		t.Error("TrimChars failed")
	}
}

func TestStringsUpperLower(t *testing.T) {
	if str.Upper("wolf") != "WOLF" {
		t.Error("Upper failed")
	}
	if str.Lower("WOLF") != "wolf" {
		t.Error("Lower failed")
	}
}

func TestStringsReplace(t *testing.T) {
	if str.Replace("hello world", "world", "wolf") != "hello wolf" {
		t.Error("Replace failed")
	}
	if str.ReplaceFirst("a-a-a", "a", "b") != "b-a-a" {
		t.Error("ReplaceFirst failed")
	}
}

func TestStringsLength(t *testing.T) {
	if str.Length("wolf") != 4 {
		t.Error("Length failed")
	}
	if str.Length("") != 0 {
		t.Error("Empty length should be 0")
	}
}

func TestStringsReverse(t *testing.T) {
	if str.Reverse("wolf") != "flow" {
		t.Error("Reverse failed")
	}
}

func TestStringsIndex(t *testing.T) {
	if str.Index("hello wolf", "wolf") != 6 {
		t.Error("Index failed")
	}
	if str.Index("hello", "xyz") != -1 {
		t.Error("Expected -1 for not found")
	}
}

func TestStringsSubstring(t *testing.T) {
	if str.Substring("hello wolf", 6, 10) != "wolf" {
		t.Error("Substring failed")
	}
	if str.Substring("abc", 5, 10) != "" {
		t.Error("Expected empty for out of range")
	}
}

func TestStringsPad(t *testing.T) {
	if str.PadLeft("42", 5, "0") != "00042" {
		t.Errorf("PadLeft: got '%s'", str.PadLeft("42", 5, "0"))
	}
	if str.PadRight("hi", 5, ".") != "hi..." {
		t.Errorf("PadRight: got '%s'", str.PadRight("hi", 5, "."))
	}
}

func TestStringsIsEmpty(t *testing.T) {
	if !str.IsEmpty("") {
		t.Error("Empty string should be empty")
	}
	if !str.IsEmpty("   ") {
		t.Error("Whitespace should be empty")
	}
	if str.IsEmpty("wolf") {
		t.Error("Non-empty should not be empty")
	}
}

func TestStringsIsNumeric(t *testing.T) {
	if !str.IsNumeric("12345") {
		t.Error("Digits should be numeric")
	}
	if str.IsNumeric("12.5") {
		t.Error("Decimal should not be numeric (digits only)")
	}
	if str.IsNumeric("") {
		t.Error("Empty should not be numeric")
	}
}

func TestStringsIsAlpha(t *testing.T) {
	if !str.IsAlpha("wolf") {
		t.Error("Letters should be alpha")
	}
	if str.IsAlpha("wolf123") {
		t.Error("Mixed should not be alpha")
	}
}

func TestStringsRepeat(t *testing.T) {
	if str.Repeat("ab", 3) != "ababab" {
		t.Error("Repeat failed")
	}
}

// ========== WolfJSON Tests ==========

var js WolfJSON

func TestJSONEncodeDecode(t *testing.T) {
	data := map[string]interface{}{"name": "Wolf", "version": 1.0}
	encoded := js.Encode(data)
	if encoded == "" {
		t.Error("Expected non-empty JSON")
	}

	decoded := js.Decode(encoded)
	m, ok := decoded.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map")
	}
	if m["name"] != "Wolf" {
		t.Error("Expected 'Wolf'")
	}
}

func TestJSONDecodeMap(t *testing.T) {
	m := js.DecodeMap(`{"key": "value"}`)
	if m == nil {
		t.Fatal("Expected map, got nil")
	}
	if m["key"] != "value" {
		t.Error("Expected 'value'")
	}
}

func TestJSONDecodeArray(t *testing.T) {
	arr := js.DecodeArray(`[1, 2, 3]`)
	if arr == nil {
		t.Fatal("Expected array, got nil")
	}
	if len(arr) != 3 {
		t.Errorf("Expected 3, got %d", len(arr))
	}
}

func TestJSONEncodePretty(t *testing.T) {
	data := map[string]string{"a": "1"}
	pretty := js.EncodePretty(data)
	if pretty == "" {
		t.Fatal("Expected pretty JSON, got empty string")
	}
	if !containsStr(pretty, "\n") {
		t.Error("Pretty should have newlines")
	}
}

func TestJSONValid(t *testing.T) {
	if !js.Valid(`{"valid": true}`) {
		t.Error("Should be valid")
	}
	if js.Valid(`{invalid`) {
		t.Error("Should be invalid")
	}
}

func TestJSONDecodeError(t *testing.T) {
	res := js.Decode("{bad json")
	if res != nil {
		t.Error("Expected nil on bad json")
	}
}

// ========== WolfMath Tests ==========

var m WolfMath

func TestMathAbs(t *testing.T) {
	if m.Abs(-5.0) != 5.0 {
		t.Error("Abs failed")
	}
	if m.Abs(5.0) != 5.0 {
		t.Error("Abs positive failed")
	}
}

func TestMathCeilFloor(t *testing.T) {
	if m.Ceil(4.1) != 5.0 {
		t.Error("Ceil failed")
	}
	if m.Floor(4.9) != 4.0 {
		t.Error("Floor failed")
	}
}

func TestMathRound(t *testing.T) {
	if m.Round(4.5) != 5.0 {
		t.Error("Round failed")
	}
	if m.RoundTo(3.14159, 2) != 3.14 {
		t.Error("RoundTo failed")
	}
}

func TestMathSqrt(t *testing.T) {
	if m.Sqrt(16.0) != 4.0 {
		t.Error("Sqrt failed")
	}
}

func TestMathPow(t *testing.T) {
	if m.Pow(2.0, 10.0) != 1024.0 {
		t.Error("Pow failed")
	}
}

func TestMathLog(t *testing.T) {
	if math.Abs(m.Log(math.E)-1.0) > 0.0001 {
		t.Error("Log(e) should be 1")
	}
	if m.Log10(100.0) != 2.0 {
		t.Error("Log10(100) should be 2")
	}
}

func TestMathMinMax(t *testing.T) {
	if m.Min(3.0, 7.0) != 3.0 {
		t.Error("Min failed")
	}
	if m.Max(3.0, 7.0) != 7.0 {
		t.Error("Max failed")
	}
}

func TestMathClamp(t *testing.T) {
	if m.Clamp(15.0, 0.0, 10.0) != 10.0 {
		t.Error("Clamp above max")
	}
	if m.Clamp(-5.0, 0.0, 10.0) != 0.0 {
		t.Error("Clamp below min")
	}
	if m.Clamp(5.0, 0.0, 10.0) != 5.0 {
		t.Error("Clamp within range")
	}
}

func TestMathRandom(t *testing.T) {
	val := m.Random()
	if val < 0.0 || val >= 1.0 {
		t.Errorf("Random out of range: %f", val)
	}
}

func TestMathRandomInt(t *testing.T) {
	for i := 0; i < 50; i++ {
		val := m.RandomInt(10)
		if val < 0 || val >= 10 {
			t.Errorf("RandomInt out of range: %d", val)
		}
	}
}

func TestMathRandomRange(t *testing.T) {
	for i := 0; i < 50; i++ {
		val := m.RandomRange(5, 10)
		if val < 5 || val > 10 {
			t.Errorf("RandomRange out of range: %d", val)
		}
	}
}

func TestMathTrig(t *testing.T) {
	if math.Abs(m.Sin(0.0)) > 0.0001 {
		t.Error("Sin(0) should be 0")
	}
	if math.Abs(m.Cos(0.0)-1.0) > 0.0001 {
		t.Error("Cos(0) should be 1")
	}
}

func TestMathConstants(t *testing.T) {
	if Pi != math.Pi {
		t.Error("Pi mismatch")
	}
	if E != math.E {
		t.Error("E mismatch")
	}
}

// ========== WolfEnv Tests ==========

var env WolfEnv

func TestEnvGetSet(t *testing.T) {
	os.Setenv("WOLF_TEST_VAR", "hello")
	defer os.Unsetenv("WOLF_TEST_VAR")

	if env.Get("WOLF_TEST_VAR") != "hello" {
		t.Error("Get failed")
	}
}

func TestEnvGetDefault(t *testing.T) {
	if env.GetDefault("NONEXISTENT_WOLF_VAR", "fallback") != "fallback" {
		t.Error("GetDefault failed")
	}
}

func TestEnvSet(t *testing.T) {
	env.Set("WOLF_SET_TEST", "value")
	defer os.Unsetenv("WOLF_SET_TEST")
	if os.Getenv("WOLF_SET_TEST") != "value" {
		t.Error("Set failed")
	}
}

func TestEnvUnset(t *testing.T) {
	os.Setenv("WOLF_UNSET_TEST", "val")
	env.Unset("WOLF_UNSET_TEST")
	if os.Getenv("WOLF_UNSET_TEST") != "" {
		t.Error("Unset failed")
	}
}

func TestEnvHas(t *testing.T) {
	os.Setenv("WOLF_HAS_TEST", "yes")
	defer os.Unsetenv("WOLF_HAS_TEST")
	if !env.Has("WOLF_HAS_TEST") {
		t.Error("Has should return true")
	}
	if env.Has("WOLF_NO_EXIST_XYZ") {
		t.Error("Has should return false")
	}
}

func TestEnvAll(t *testing.T) {
	all := env.All()
	if len(all) == 0 {
		t.Error("All should return env vars")
	}
	// PATH should exist
	if _, ok := all["PATH"]; !ok {
		t.Error("Expected PATH in environment")
	}
}

func TestEnvLoadFile(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	content := `# Database config
DB_HOST=localhost
DB_PORT=3306
DB_NAME="wolfdb"
EMPTY_VAR=
QUOTED='single quoted'
`
	os.WriteFile(envFile, []byte(content), 0644)

	err := env.LoadFile(envFile)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if os.Getenv("DB_HOST") != "localhost" {
		t.Error("Expected DB_HOST=localhost")
	}
	if os.Getenv("DB_PORT") != "3306" {
		t.Error("Expected DB_PORT=3306")
	}
	if os.Getenv("DB_NAME") != "wolfdb" {
		t.Error("Expected DB_NAME=wolfdb (unquoted)")
	}
	if os.Getenv("QUOTED") != "single quoted" {
		t.Error("Expected unquoted value")
	}

	// Cleanup
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("EMPTY_VAR")
	os.Unsetenv("QUOTED")
}

func TestEnvLoadFileMissing(t *testing.T) {
	err := env.LoadFile("/nonexistent/.env")
	if err == nil {
		t.Error("Expected error for missing file")
	}
}
