package deploy_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDockerfileStory202Requirements(t *testing.T) {
	t.Parallel()

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(root, "Dockerfile"))
	if err != nil {
		t.Fatalf("read dockerfile: %v", err)
	}
	content := string(raw)
	upper := strings.ToUpper(content)

	if !strings.Contains(upper, "FROM ") || !strings.Contains(upper, " AS BUILDER") {
		t.Fatalf("dockerfile must use multi-stage build")
	}
	if !strings.Contains(content, "FROM golang:1.26.4-alpine AS builder") {
		t.Fatalf("builder stage must use the supported Go 1.26.4 alpine image")
	}
	if !strings.Contains(content, "github.com/a-h/templ/cmd/templ@v0.3.1020") {
		t.Fatalf("builder stage must install the pinned templ generator")
	}
	if !strings.Contains(upper, "FROM ALPINE") {
		t.Fatalf("dockerfile must have alpine runtime stage")
	}
	if !strings.Contains(upper, "COPY --FROM=BUILDER") {
		t.Fatalf("runtime image must copy artifact from builder stage")
	}
	runtime := afterLastFrom(upper)
	if strings.Contains(runtime, " GO BUILD") || strings.Contains(runtime, " GO TEST") || strings.Contains(runtime, " GO RUN") || strings.Contains(runtime, " GO INSTALL") {
		t.Fatalf("runtime image must not include go toolchain usage")
	}
	if !strings.Contains(runtime, "USER CALDO") {
		t.Fatalf("runtime image must run as non-root user")
	}
	if !strings.Contains(runtime, "EXPOSE 8080") {
		t.Fatalf("runtime image must expose only port 8080")
	}
	if !strings.Contains(upper, "VOLUME [\"/DATA\"]") {
		t.Fatalf("runtime image must declare /data as persistent volume")
	}
	if !regexp.MustCompile(`(?m)^RUN\s+.*(APK\s+ADD|ADDUSER)`).MatchString(runtime) {
		t.Fatalf("runtime stage must include setup commands")
	}
	if !strings.Contains(runtime, "WGET") {
		t.Fatalf("image must provide a healthcheck-capable tool like wget")
	}
	staticCopy := strings.Index(runtime, "COPY WEB/STATIC /APP/WEB/STATIC")
	staticChmod := strings.Index(runtime, "CHMOD -R A+RX /APP/WEB/STATIC")
	runtimeUser := strings.Index(runtime, "USER CALDO")
	if staticCopy == -1 {
		t.Fatalf("runtime image must copy static assets")
	}
	if staticChmod == -1 {
		t.Fatalf("runtime image must make static assets readable by the non-root user")
	}
	if runtimeUser == -1 || staticCopy > staticChmod || staticChmod > runtimeUser {
		t.Fatalf("runtime image must make static assets readable before switching to non-root user")
	}
}

func TestBuildAssetsPublishesReadableStaticFiles(t *testing.T) {
	t.Parallel()

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(root, "scripts", "build-assets.sh"))
	if err != nil {
		t.Fatalf("read asset build script: %v", err)
	}
	content := string(raw)

	if !strings.Contains(content, "chmod 0644 \"$source_path\"") {
		t.Fatalf("asset build script must normalize generated file permissions before publishing")
	}
	for _, expected := range []string{
		"publish_static_file \"$tmp_path\" \"$target_path\"",
		"publish_static_file \"$tmp_css\" \"$STATIC_DIR/$target_name\"",
		"publish_static_file \"$tmp_manifest\" \"$STATIC_DIR/manifest.json\"",
		"chmod 0644 \"$STATIC_DIR/$target_name\"",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("asset build script must publish readable files; missing %q", expected)
		}
	}
	for _, unsafeMove := range []string{
		"mv \"$tmp_path\" \"$target_path\"",
		"mv \"$tmp_css\" \"$STATIC_DIR/$target_name\"",
		"mv \"$tmp_manifest\" \"$STATIC_DIR/manifest.json\"",
	} {
		if strings.Contains(content, unsafeMove) {
			t.Fatalf("asset build script must not publish mktemp files without normalizing permissions: %q", unsafeMove)
		}
	}
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func afterLastFrom(content string) string {
	idx := strings.LastIndex(content, "FROM ")
	if idx == -1 {
		return content
	}
	return content[idx:]
}
