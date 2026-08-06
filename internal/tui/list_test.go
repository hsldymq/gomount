package tui

import (
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

func TestEntryAddrWebDAV(t *testing.T) {
	entry := config.MountEntry{
		Type:   "webdav",
		WebDAV: &config.WebDAVConfig{URL: "https://cloud.example.com/dav", Path: "/team"},
	}

	if got := entryAddr(entry); got != "https://cloud.example.com/dav:/team" {
		t.Fatalf("entryAddr() = %q", got)
	}
}

func TestEntryAddrWebDAVRedactsURLUserinfo(t *testing.T) {
	entry := config.MountEntry{
		Type:   "webdav",
		WebDAV: &config.WebDAVConfig{URL: "https://user:secret@cloud.example.com/dav", Path: "/team"},
	}

	if got := entryAddr(entry); got != "https://xxxxx@cloud.example.com/dav:/team" {
		t.Fatalf("entryAddr() = %q", got)
	}
}

func TestEntryAddrS3(t *testing.T) {
	entry := config.MountEntry{
		Type: "s3",
		S3:   &config.S3Config{Provider: "aliyun_oss", Bucket: "example-bucket", Path: "/backups/", Endpoint: "oss-cn-hangzhou.aliyuncs.com"},
	}

	if got := entryAddr(entry); got != "s3://example-bucket/backups@oss-cn-hangzhou.aliyuncs.com" {
		t.Fatalf("entryAddr() = %q", got)
	}
}

func TestEntrySegmentsS3IncludesProvider(t *testing.T) {
	entry := config.MountEntry{
		Type: "s3",
		S3:   &config.S3Config{Provider: "aliyun_oss", Bucket: "example-bucket"},
	}

	segments := entrySegments(entry)
	if got := segments[len(segments)-1].Text; got != "(s3: aliyun_oss)" {
		t.Fatalf("type label = %q", got)
	}
}
