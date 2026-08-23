package knowledge

import (
	"os"
	"testing"
)

func TestSampleMemoryFixture_SplitFrontmatter(t *testing.T) {
	t.Parallel()

	md, err := os.ReadFile("testdata/sample_memory.md")
	if err != nil {
		t.Fatalf("read testdata/sample_memory.md: %v", err)
	}

	// Only SplitFrontmatter — keep this branch independent of Task 41 ParseWikilinks.
	fm, _, err := SplitFrontmatter(string(md))
	if err != nil {
		t.Fatalf("SplitFrontmatter: %v", err)
	}
	if fm.Title == "" {
		t.Fatal("title must be non-empty")
	}
}
