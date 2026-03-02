# Standard Library 🐺

Wolf comes with a powerful, embedded standard library that acts as a wrapper over Go's core standard libraries (`strings`, `math`, `encoding/json`, `os`, etc.). Functions are accessed via namespaces (e.g., `Wolf\Strings.Contains()`).

---

## Wolf\Strings

Provides 22 string manipulation functions.

| Function | Signature | Description |
|----------|-----------|-------------|
| `Contains` | `(s, substr string) bool` | Checks if string contains substring. |
| `StartsWith` | `(s, prefix string) bool` | Checks if string starts with prefix. |
| `EndsWith` | `(s, suffix string) bool` | Checks if string ends with suffix. |
| `Split` | `(s, sep string) []string` | Divides string into an array. |
| `Join` | `(parts []string, sep string) string` | Concatenates an array into a string. |
| `Trim` | `(s string) string` | Removes bounding whitespace. |
| `TrimChars` | `(s, chars string) string` | Removes specific bounding characters. |
| `Upper` | `(s string) string` | Converts to uppercase. |
| `Lower` | `(s string) string` | Converts to lowercase. |
| `Replace` | `(s, old, new string) string` | Replaces all occurrences. |
| `ReplaceFirst` | `(s, old, new string) string` | Replaces first occurrence. |
| `Length` | `(s string) int` | Analyzes character length. |
| `Repeat` | `(s string, count int) string` | Repeats string `count` times. |
| `Reverse` | `(s string) string` | Reverses characters in string. |
| `Index` | `(s, substr string) int` | Returns index, or -1 if missing. |
| `Substring` | `(s string, start, end int) string` | Extracts slice of string. |
| `PadLeft` | `(s string, len int, pad string) string` | Left-pads string to required length. |
| `PadRight`| `(s string, len int, pad string) string` | Right-pads string to required length. |
| `IsEmpty` | `(s string) bool` | Checks if empty or whitespace-only. |
| `IsNumeric` | `(s string) bool` | Checks if string only contains digits. |
| `IsAlpha` | `(s string) bool` | Checks if string only contains letters. |
| `Title` | `(s string) string` | Capitalizes words in string. |

**Usage:**
```wolf
$is_wolf = Wolf\Strings.Contains("Red Wolf", "Wolf")
```

---

## Wolf\JSON

Provides 6 JSON serialization functions.

| Function | Signature | Description |
|----------|-----------|-------------|
| `Encode` | `(v interface{}) (string, error)` | Converts map/array/value to JSON string. |
| `Decode` | `(s string) (interface{}, error)` | Parses JSON string into generic interface. |
| `DecodeMap`| `(s string) (map[string]interface{}, error)`| Parses JSON object string into Map. |
| `DecodeArray`|`(s string) ([]interface{}, error)`| Parses JSON array string into Array. |
| `EncodePretty`|`(v interface{}) (string, error)` | Pretty-prints JSON payload. |
| `Valid` | `(s string) bool` | Validates JSON format strictly. |

**Usage:**
```wolf
$json_str = Wolf\JSON.Encode({"id": 1, "name": "Ghost"})
```

---

## Wolf\Math

Provides 20 mathematical functions and 2 constants (`Pi`, `E`).

| Function | Signature | Description |
|----------|-----------|-------------|
| `Abs` | `(x float64) float64` | Absolute value. |
| `Ceil` | `(x float64) float64` | Round up. |
| `Floor` | `(x float64) float64` | Round down. |
| `Round` | `(x float64) float64` | Round to integer. |
| `RoundTo`| `(x float64, places int) float64`| Round to *n* decimal places. |
| `Sqrt` | `(x float64) float64` | Square root. |
| `Pow` | `(x, y float64) float64` | Exponentiation (x^y). |
| `Log` | `(x float64) float64` | Natural logarithm. |
| `Min` | `(a, b float64) float64` | Smaller of two inputs. |
| `Max` | `(a, b float64) float64` | Larger of two inputs. |
| `Clamp` | `(val, min, max float64) float64`| Restrict value between bounds. |
| `Random` | `() float64` | Random float in [0.0, 1.0). |
| `RandomInt`|`(max int) int` | Random integer in [0, max). |
| `RandomRange`|`(min, max int) int`| Random integer in [min, max]. |
| `Sin`, `Cos`, `Tan` | `(x float64) float64` | Trigonometry. |

**Usage:**
```wolf
$distance = Wolf\Math.Sqrt(16.0)
$clamped = Wolf\Math.Clamp($val, 0.0, 100.0)
```

---

## Wolf\Env 

Provides 8 environment manipulation utilities, including `.env` file reading.

| Function | Signature | Description |
|----------|-----------|-------------|
| `Get` | `(key string) string` | Fetches Env Var or empty string. |
| `GetDefault`| `(key, def string) string`| Fetches Env Var with fallback. |
| `Set` | `(key, value string) error` | Sets system Env Var. |
| `Unset` | `(key string) error` | Deletes system Env Var. |
| `All` | `() map[string]string` | Dumps Env Vars into Wolf Map. |
| `Has` | `(key string) bool` | Checks if Env Var exists. |
| `LoadFile`| `(path string) error` | Parses .env file safely. |
| `Require` | `(key string) string` | Fetches Env Var or Panics. |

**Usage:**
```wolf
Wolf\Env.LoadFile(".env")
$api_key = Wolf\Env.Require("OPENAI_API_KEY")
```
