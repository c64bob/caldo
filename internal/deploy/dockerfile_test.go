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
	if !strings.Contains(upper, "USER CALDO") {
		t.Fatalf("runtime image must run as non-root user")
	}
	if !strings.Contains(upper, "EXPOSE 8080") {
		t.Fatalf("runtime image must expose only port 8080")
	}
	if !strings.Contains(upper, "VOLUME [\"/DATA\"]") {
		t.Fatalf("runtime image must declare /data as persistent volume")
	}
	if !regexp.MustCompile(`(?m)^RUN\s+.*(APK\s+ADD|ADDUSER)`).MatchString(upper) {
		t.Fatalf("runtime stage must include setup commands")
	}
	if !strings.Contains(upper, "WGET") {
		t.Fatalf("image must provide a healthcheck-capable tool like wget")
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
