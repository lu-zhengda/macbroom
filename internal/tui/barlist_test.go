package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lu-zhengda/macbroom/internal/scanner"
)

func TestRenderBarList_Basic(t *testing.T) {
	nodes := []scanner.SpaceLensNode{
		{Name: "Toolchains", Size: 3700000000, IsDir: true, Path: "/a"},
		{Name: "Platforms", Size: 3400000000, IsDir: true, Path: "/b"},
		{Name: "usr", Size: 325800000, IsDir: true, Path: "/c"},
	}

	result := renderBarList(nodes, 60, 10, 0, 0)

	if result == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(result, "Toolchains") {
		t.Error("expected Toolchains in output")
	}
	if !strings.Contains(result, "Platforms") {
		t.Error("expected Platforms in output")
	}
	if !strings.Contains(result, "usr") {
		t.Error("expected usr in output")
	}
}

func TestRenderBarList_Empty(t *testing.T) {
	result := renderBarList(nil, 60, 10, 0, 0)
	if !strings.Contains(result, "Empty") {
		t.Error("expected empty message")
	}
}

func TestRenderBarList_SelectedHighlight(t *testing.T) {
	nodes := []scanner.SpaceLensNode{
		{Name: "dir1", Size: 1000, IsDir: true, Path: "/a"},
		{Name: "dir2", Size: 500, IsDir: true, Path: "/b"},
	}

	// Cursor on item 0.
	result0 := renderBarList(nodes, 60, 10, 0, 0)
	if !strings.Contains(result0, ">") {
		t.Error("expected '>' cursor marker for selected item")
	}

	// Cursor on item 1.
	result1 := renderBarList(nodes, 60, 10, 1, 0)
	if result1 == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestRenderBarList_Scrolling(t *testing.T) {
	var nodes []scanner.SpaceLensNode
	for i := 0; i < 20; i++ {
		nodes = append(nodes, scanner.SpaceLensNode{
			Name:  fmt.Sprintf("dir%d", i),
			Size:  int64(1000 - i*10),
			IsDir: true,
			Path:  fmt.Sprintf("/d%d", i),
		})
	}

	// Only 5 visible lines, scroll offset 5.
	result := renderBarList(nodes, 60, 5, 7, 5)
	if !strings.Contains(result, "dir5") {
		t.Error("expected dir5 in scrolled output")
	}
	if strings.Contains(result, "dir0") {
		t.Error("dir0 should not appear when scrolled past")
	}
	// Should show scroll indicator.
	if !strings.Contains(result, "of 20") {
		t.Error("expected scroll indicator")
	}
}

func TestRenderBarList_DirSuffix(t *testing.T) {
	nodes := []scanner.SpaceLensNode{
		{Name: "mydir", Size: 1000, IsDir: true, Path: "/a"},
		{Name: "myfile.txt", Size: 500, IsDir: false, Path: "/b"},
	}

	result := renderBarList(nodes, 60, 10, 0, 0)
	if !strings.Contains(result, "mydir/") {
		t.Error("expected trailing slash for directory")
	}
	if strings.Contains(result, "myfile.txt/") {
		t.Error("file should not have trailing slash")
	}
}

func TestRenderBarList_NarrowWidth(t *testing.T) {
	nodes := []scanner.SpaceLensNode{
		{Name: "dir1", Size: 1000, IsDir: true, Path: "/a"},
	}

	result := renderBarList(nodes, 30, 10, 0, 0)
	if result == "" {
		t.Fatal("expected non-empty output even at narrow width")
	}
}
