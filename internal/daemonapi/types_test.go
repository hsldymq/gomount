package daemonapi

import "testing"

func TestManagedTypesContainsWebDAVOnly(t *testing.T) {
	types := ManagedTypes()
	if len(types) != 1 || types[0] != "webdav" {
		t.Fatalf("unexpected managed types: %v", types)
	}
}

func TestEntrySnapshotValidateRequiresWebDAVURL(t *testing.T) {
	entry := EntrySnapshot{
		Name:         "docs",
		Type:         "webdav",
		MountDirPath: "/mnt/docs",
	}

	if err := entry.Validate(); err == nil {
		t.Fatal("expected missing url to fail validation")
	}
}

func TestEntrySnapshotValidateAcceptsWebDAVURL(t *testing.T) {
	entry := EntrySnapshot{
		Name:         "docs",
		Type:         "webdav",
		MountDirPath: "/mnt/docs",
		Source:       Source{URL: "https://cloud.example.com/dav"},
	}

	if err := entry.Validate(); err != nil {
		t.Fatalf("expected valid webdav source, got %v", err)
	}
}
