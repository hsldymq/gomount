//go:build darwin && cgo && cmount

package daemon

// On macOS rclone registers cmount under the standard "mount" method name.
import _ "github.com/rclone/rclone/cmd/cmount"
