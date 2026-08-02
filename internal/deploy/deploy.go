package deploy

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"wolf/internal/compiler"
)

func Run(c *compiler.Compiler, target string, sourceFile string) error {
	switch target {
	case "container":
		return deployContainer(c, sourceFile)
	case "shared":
		return deployShared(c, sourceFile)
	case "serverless":
		return fmt.Errorf("serverless deployment is not supported")
	default:
		return fmt.Errorf("unsupported deploy target: %s. Supported targets are: container, shared", target)
	}
}

func deployContainer(c *compiler.Compiler, sourceFile string) error {
	fmt.Printf("Deploying %s to container...\n", sourceFile)
	
	binName := strings.TrimSuffix(filepath.Base(sourceFile), ".wolf")

	// Create multi-stage Dockerfile matching Scope 1 static build
	dockerfileContent := fmt.Sprintf(`# -----------------------------------------------------------------------------
# STAGE 1: Builder
# -----------------------------------------------------------------------------
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache zig make git

WORKDIR /app
COPY . .

RUN go build -o /usr/local/bin/wolf ./cmd/wolf
RUN wolf build --static %s

# -----------------------------------------------------------------------------
# STAGE 2: Runner (Zero-Dependency)
# -----------------------------------------------------------------------------
FROM scratch

COPY --from=builder /app/wolf_out/%s /server

EXPOSE 8080
CMD ["/server"]
`, sourceFile, binName)

	err := os.WriteFile("Dockerfile.deploy", []byte(dockerfileContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}
	
	fmt.Println("Building Docker image (this will compile a static musl binary inside Alpine)...")
	cmd := exec.Command("docker", "build", "-f", "Dockerfile.deploy", "-t", "wolf-app", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	fmt.Println("Successfully built container image 'wolf-app'.")
	return nil
}

func deployShared(c *compiler.Compiler, sourceFile string) error {
	fmt.Printf("Deploying %s to shared hosting...\n", sourceFile)
	
	sourceBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	// Shared hosting implies CGI
	c.Config.Target.CGI = true
	c.Config.Target.Mode = "api"
	
	res, err := c.Build(string(sourceBytes), sourceFile)
	if err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}
	binaryPath := res.OutputPath
	
	deployZip := "deploy.zip"
	outFile, err := os.Create(deployZip)
	if err != nil {
		return fmt.Errorf("failed to create zip: %w", err)
	}
	defer outFile.Close()
	
	w := zip.NewWriter(outFile)
	defer w.Close()
	
	// Add binary to zip
	binFile, err := os.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to open binary: %w", err)
	}
	defer binFile.Close()
	
	binName := filepath.Base(binaryPath)
	fh := &zip.FileHeader{
		Name:   binName,
		Method: zip.Deflate,
	}
	fh.SetMode(0755)
	f, err := w.CreateHeader(fh)
	if err != nil {
		return fmt.Errorf("failed to add binary to zip: %w", err)
	}
	_, err = io.Copy(f, binFile)
	if err != nil {
		return fmt.Errorf("failed to write binary to zip: %w", err)
	}
	
	// Add .htaccess for CGI routing
	htaccessContent := fmt.Sprintf(`RewriteEngine On
RewriteCond %%{REQUEST_FILENAME} !-f
RewriteRule ^(.*)$ %%s/$1 [QSA,L]
`, binName)

	hh := &zip.FileHeader{
		Name:   ".htaccess",
		Method: zip.Deflate,
	}
	hh.SetMode(0644)
	h, err := w.CreateHeader(hh)
	if err != nil {
		return fmt.Errorf("failed to add .htaccess to zip: %w", err)
	}
	_, err = h.Write([]byte(htaccessContent))
	if err != nil {
		return fmt.Errorf("failed to write .htaccess to zip: %w", err)
	}
	
	fmt.Printf("Successfully packaged %s into %s.\n", binName, deployZip)
	fmt.Println("Upload deploy.zip to your shared host and extract it in the public_html directory.")
	return nil
}
