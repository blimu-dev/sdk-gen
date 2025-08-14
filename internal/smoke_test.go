package main

import (
	"os"
	"testing"

	sdkgen "github.com/blimu-dev/sdk-gen"
)

func TestBuildIR_NoSpec(t *testing.T) {
	// Smoke: ensure binary builds and ValidateSpec errors on missing file
	if _, err := os.Stat("/no/such/file.yaml"); err == nil {
		t.Fatal("expected no file")
	}
	if err := sdkgen.ValidateSpec("/no/such/file.yaml"); err == nil {
		t.Fatal("expected error")
	}
}
