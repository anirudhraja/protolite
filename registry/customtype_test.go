package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadProto(t *testing.T, content string) (*Registry, string) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "customtype_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	protoPath := filepath.Join(tmpDir, "test.proto")
	if err := os.WriteFile(protoPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(protoPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })

	r := NewRegistry([]string{""})
	return r, protoPath
}

func TestCustomType_ParsedOnBytesField(t *testing.T) {
	content := `syntax = "proto3";
package test.customtype;

message Holder {
  bytes session = 1 [(gogoproto.customtype) = "USession"];
  string name = 2;
}
`
	r, protoPath := loadProto(t, content)
	file, err := os.Open(protoPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := r.LoadSchema(file, protoPath); err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}

	msg, err := r.GetMessage("test.customtype.Holder")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}

	var session, name bool
	for _, f := range msg.Fields {
		switch f.Name {
		case "session":
			session = true
			if !f.CustomType {
				t.Errorf("session field: CustomType = false, want true")
			}
		case "name":
			name = true
			if f.CustomType {
				t.Errorf("name field: CustomType = true, want false")
			}
		}
	}
	if !session || !name {
		t.Fatalf("expected to find both fields, got session=%v name=%v", session, name)
	}
}

func TestCustomType_RejectedOnNonBytesField(t *testing.T) {
	content := `syntax = "proto3";
package test.customtype;

message Bad {
  string session = 1 [(gogoproto.customtype) = "USession"];
}
`
	r, protoPath := loadProto(t, content)
	file, err := os.Open(protoPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	err = r.LoadSchema(file, protoPath)
	if err == nil {
		t.Fatalf("expected error for (gogoproto.customtype) on non-bytes field, got nil")
	}
	if !strings.Contains(err.Error(), "gogoproto.customtype") {
		t.Errorf("error should mention gogoproto.customtype, got: %v", err)
	}
}
