package daemonapi

import "github.com/hsldymq/gomount/internal/config"

func FromMountEntry(entry *config.MountEntry) (EntrySnapshot, bool) {
	snapshot := EntrySnapshot{
		Name:         entry.Name,
		Type:         entry.Type,
		MountDirPath: entry.MountDirPath,
		Options:      mapStringAny(entry.Options),
	}
	switch entry.Type {
	case "webdav":
		if entry.WebDAV == nil {
			return snapshot, false
		}
		snapshot.Source = Source{URL: entry.WebDAV.URL, Username: entry.WebDAV.Username, Password: entry.WebDAV.Password, Path: entry.WebDAV.Path}
	case "oss":
		if entry.OSS == nil {
			return snapshot, false
		}
		snapshot.Source = Source{
			Bucket: entry.OSS.Bucket, Path: entry.OSS.Path, Endpoint: entry.OSS.Endpoint,
			AccessKeyID: entry.OSS.AccessKeyID, AccessKeySecret: entry.OSS.AccessKeySecret,
			SecurityToken: entry.OSS.SecurityToken,
		}
	default:
		return snapshot, false
	}
	return snapshot, true
}

func mapStringAny(in map[string]interface{}) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
