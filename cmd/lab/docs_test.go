// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestReadmePointsToArticle(t *testing.T) {
	root := filepath.Join("..", "..")
	readme := filepath.Join(root, "README.md")
	info, err := os.Lstat(readme)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("README.md must be a symlink to the article, not a second walkthrough")
	}
	target, err := os.Readlink(readme)
	if err != nil {
		t.Fatal(err)
	}
	if target != "article/substack.md" {
		t.Fatalf("unexpected README target: %q", target)
	}
	article, err := os.ReadFile(filepath.Join(root, target))
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(readme)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, article) {
		t.Fatal("README.md does not resolve to the article content")
	}
	if !bytes.Contains(article, []byte(`<a name="1-try-the-counter"></a>`)) {
		t.Fatal("keep the counter anchor used by the published Substack article")
	}
}
