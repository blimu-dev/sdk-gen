package main

import (
	"os"
	"testing"

	cli "github.com/blimu-dev/sdk-gen/internal/cli"
)

func TestBuildIR_NoSpec(t *testing.T) {
	// Smoke: ensure binary builds and RunValidate errors on missing file
	if _, err := os.Stat("/no/such/file.yaml"); err == nil {
		t.Fatal("expected no file")
	}
	if err := cli.RunValidate("/no/such/file.yaml"); err == nil {
		t.Fatal("expected error")
	}
}
