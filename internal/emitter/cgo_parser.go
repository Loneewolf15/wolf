package emitter

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// CGOFunction represents an extracted function signature from a Go c-archive header.
type CGOFunction struct {
	Name       string
	ReturnType string
	Params     []string
}

// ParseCGOHeader reads the generated .h file and extracts all `extern` function signatures.
func ParseCGOHeader(headerPath string) ([]CGOFunction, error) {
	file, err := os.Open(headerPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var funcs []CGOFunction

	// Simplified regex to capture: extern <returnType> <name>(<params>);
	// e.g. extern char* WolfML_Exec(char* src, char* inJson);
	externRegex := regexp.MustCompile(`(?m)^extern\s+([A-Za-z0-9_* ]+?)\s+([A-Za-z0-9_]+)\((.*?)\);`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "extern ") {
			matches := externRegex.FindStringSubmatch(line)
			if len(matches) == 4 {
				retType := strings.TrimSpace(matches[1])
				name := strings.TrimSpace(matches[2])
				paramsRaw := strings.Split(matches[3], ",")

				var params []string
				for _, p := range paramsRaw {
					p = strings.TrimSpace(p)
					if p != "" && p != "void" {
						// Extract just the type (everything before the last space)
						parts := strings.Fields(p)
						if len(parts) > 1 {
							// For "char* src", we want "char*"
							// For "long long unsigned int x", we want "long long unsigned int"
							params = append(params, strings.Join(parts[:len(parts)-1], " "))
						} else {
							params = append(params, p)
						}
					}
				}

				funcs = append(funcs, CGOFunction{
					Name:       name,
					ReturnType: retType,
					Params:     params,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return funcs, nil
}
