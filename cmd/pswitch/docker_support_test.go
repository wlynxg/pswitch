package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerfileUsesPersistentDataDirectory(t *testing.T) {
	path := filepath.Join("..", "..", "Dockerfile")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	text := string(content)
	for _, snippet := range []string{
		"WORKDIR /data",
		"EXPOSE 8080",
		`CMD ["--config", "/data/config.toml"]`,
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("Dockerfile missing %q", snippet)
		}
	}
}

func TestDockerComposePersistsRuntimeFiles(t *testing.T) {
	path := filepath.Join("..", "..", "docker-compose.yml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	text := string(content)
	for _, snippet := range []string{
		`- "8080:8080"`,
		"./data:/data",
		"PSWITCH_ADMIN_TOKEN",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("docker-compose.yml missing %q", snippet)
		}
	}
}

func TestCIWorkflowBuildsDockerImage(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "ci.yml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	text := string(content)
	for _, snippet := range []string{
		"docker build",
		"Docker image",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("ci workflow missing %q", snippet)
		}
	}
}
