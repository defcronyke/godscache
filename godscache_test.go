package godscache

import "testing"

func TestNewGodscache(t *testing.T) {
	g := NewGodscache()
	if g == nil {
		t.Fatalf("Instantiating new Godscache struct failed")
	}
}
