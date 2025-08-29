package gnmi

import (
	"fmt"
	"strings"

	"github.com/openconfig/gnmi/proto/gnmi"
)

// pathToString converts a gNMI path to a string representation for logging and debugging.
// It handles path elements with keys (e.g., filesystem[path=/host]).
func pathToString(path *gnmi.Path) string {
	if path == nil {
		return "/"
	}

	parts := make([]string, 0, len(path.Elem))
	for _, elem := range path.Elem {
		part := elem.Name
		if len(elem.Key) > 0 {
			// Add keys if present (e.g., [path=/host])
			var keys []string
			for k, v := range elem.Key {
				keys = append(keys, fmt.Sprintf("%s=%s", k, v))
			}
			part += "[" + strings.Join(keys, ",") + "]"
		}
		parts = append(parts, part)
	}

	return "/" + strings.Join(parts, "/")
}

// isFilesystemPath checks if the given path is requesting filesystem information.
// Returns true if the path starts with /sonic/system/filesystem.
func isFilesystemPath(path *gnmi.Path) bool {
	return len(path.Elem) >= 3 &&
		path.Elem[0].Name == "sonic" &&
		path.Elem[1].Name == "system" &&
		path.Elem[2].Name == "filesystem"
}

// extractFilesystemPath extracts the filesystem path from a gNMI path.
// For example, from /sonic/system/filesystem[path=/host]/disk-space,
// it extracts "/host".
func extractFilesystemPath(path *gnmi.Path) (string, error) {
	if !isFilesystemPath(path) {
		return "", fmt.Errorf("not a filesystem path: %s", pathToString(path))
	}

	fsPath, ok := path.Elem[2].Key["path"]
	if !ok {
		return "", fmt.Errorf("filesystem path not specified, expected format: /sonic/system/filesystem[path=<path>]/...")
	}

	return fsPath, nil
}

// isDiskSpacePath checks if the path is requesting disk space information.
// Returns true if the path contains /disk-space.
func isDiskSpacePath(path *gnmi.Path) bool {
	return isFilesystemPath(path) &&
		len(path.Elem) >= 4 &&
		path.Elem[3].Name == "disk-space"
}

// getDiskSpaceField determines which disk space field is being requested.
// Returns "both", "total", or "available" based on the path.
func getDiskSpaceField(path *gnmi.Path) (string, error) {
	if !isDiskSpacePath(path) {
		return "", fmt.Errorf("not a disk space path: %s", pathToString(path))
	}

	// If path ends at disk-space, return both metrics
	if len(path.Elem) == 4 {
		return "both", nil
	}

	// If path has a specific field, check what it is
	if len(path.Elem) == 5 {
		switch path.Elem[4].Name {
		case "total-mb":
			return "total", nil
		case "available-mb":
			return "available", nil
		default:
			return "", fmt.Errorf("unknown disk space metric: %s", path.Elem[4].Name)
		}
	}

	return "", fmt.Errorf("invalid disk space path structure: %s", pathToString(path))
}

// isFirmwarePath checks if the given path is requesting firmware file information.
// Returns true if the path starts with /sonic/system/firmware.
func isFirmwarePath(path *gnmi.Path) bool {
	return len(path.Elem) >= 3 &&
		path.Elem[0].Name == "sonic" &&
		path.Elem[1].Name == "system" &&
		path.Elem[2].Name == "firmware"
}

// extractFirmwareDirectory extracts the firmware directory path from a gNMI path.
// For example, from /sonic/system/firmware[directory=/lib/firmware]/files,
// it extracts "/lib/firmware".
func extractFirmwareDirectory(path *gnmi.Path) (string, error) {
	if !isFirmwarePath(path) {
		return "", fmt.Errorf("not a firmware path: %s", pathToString(path))
	}

	firmwareDir, ok := path.Elem[2].Key["directory"]
	if !ok {
		return "", fmt.Errorf("firmware directory not specified, expected format: " +
			"/sonic/system/firmware[directory=<path>]/...")
	}

	return firmwareDir, nil
}

// isFirmwareFilesPath checks if the path is requesting firmware files listing.
// Returns true if the path contains /files.
func isFirmwareFilesPath(path *gnmi.Path) bool {
	return isFirmwarePath(path) &&
		len(path.Elem) >= 4 &&
		path.Elem[3].Name == "files"
}

// getFirmwareFileField determines which firmware file field is being requested.
// Returns "list", "count", or specific filename based on the path.
func getFirmwareFileField(path *gnmi.Path) (string, error) {
	if !isFirmwareFilesPath(path) {
		return "", fmt.Errorf("not a firmware files path: %s", pathToString(path))
	}

	// If path ends at files, return list of all files
	if len(path.Elem) == 4 {
		return "list", nil
	}

	// If path has a specific field, check what it is
	if len(path.Elem) == 5 {
		switch path.Elem[4].Name {
		case "count":
			return "count", nil
		default:
			// Assume it's a specific filename
			return path.Elem[4].Name, nil
		}
	}

	return "", fmt.Errorf("invalid firmware files path structure: %s", pathToString(path))
}
