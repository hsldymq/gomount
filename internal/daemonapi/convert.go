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
	case "s3":
		if entry.S3 == nil {
			return snapshot, false
		}
		snapshot.Source = Source{
			Provider: entry.S3.Provider, Bucket: entry.S3.Bucket, Path: entry.S3.Path,
			Region: entry.S3.Region, Endpoint: entry.S3.Endpoint,
			AccessKeyID: entry.S3.AccessKeyID, SecretAccessKey: entry.S3.SecretAccessKey,
			SessionToken: entry.S3.SessionToken, EnvAuth: entry.S3.EnvAuth,
			ForcePathStyle: entry.S3.ForcePathStyle,
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
