package daemonapi

import "testing"

func TestManagedTypesContainsWebDAVAndOSS(t *testing.T) {
	types := ManagedTypes()
	if len(types) != 2 || types[0] != "webdav" || types[1] != "aliyun_oss" {
		t.Fatalf("unexpected managed types: %v", types)
	}
}

func TestEntrySnapshotValidateAcceptsOSS(t *testing.T) {
	entry := EntrySnapshot{Name: "archive", Type: "aliyun_oss", MountDirPath: "/mnt/oss", Source: Source{
		Bucket: "my-bucket", Endpoint: "oss-cn-hangzhou.aliyuncs.com", AccessKeyID: "id", AccessKeySecret: "secret",
	}}
	if err := entry.Validate(); err != nil {
		t.Fatalf("expected valid oss source, got %v", err)
	}
}

func TestEntrySnapshotValidateRejectsIncompleteOSS(t *testing.T) {
	entry := EntrySnapshot{Name: "archive", Type: "aliyun_oss", MountDirPath: "/mnt/oss", Source: Source{Bucket: "my-bucket"}}
	if err := entry.Validate(); err == nil {
		t.Fatal("expected incomplete oss source to fail validation")
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
