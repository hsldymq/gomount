package daemonapi

import "testing"

func TestManagedTypesContainsWebDAVAndS3(t *testing.T) {
	types := ManagedTypes()
	if len(types) != 2 || types[0] != "webdav" || types[1] != "s3" {
		t.Fatalf("unexpected managed types: %v", types)
	}
}

func TestEntrySnapshotValidateAcceptsS3(t *testing.T) {
	entry := EntrySnapshot{Name: "archive", Type: "s3", MountDirPath: "/mnt/s3", Source: Source{
		Provider: "aws", Bucket: "my-bucket", AccessKeyID: "id", SecretAccessKey: "secret",
	}}
	if err := entry.Validate(); err != nil {
		t.Fatalf("expected valid s3 source, got %v", err)
	}
}

func TestEntrySnapshotValidateRejectsIncompleteS3(t *testing.T) {
	entry := EntrySnapshot{Name: "archive", Type: "s3", MountDirPath: "/mnt/s3", Source: Source{Provider: "aws"}}
	if err := entry.Validate(); err == nil {
		t.Fatal("expected incomplete s3 source to fail validation")
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
