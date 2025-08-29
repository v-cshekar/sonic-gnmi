// Package file implements the gNOI File service for SONiC systems.
//
// This service provides file management operations including:
//   - Firmware file listing from specified directories
//   - File transfer capabilities (planned)
//   - File metadata and status operations (planned)
//
// The service integrates with the existing gNMI infrastructure to provide
// a unified interface for file system operations.
package file

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openconfig/gnoi/file"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the gNOI File service for SONiC systems.
type Server struct {
	file.UnimplementedFileServer
	rootFS string
}

// NewServer creates a new gNOI File server instance.
func NewServer(rootFS string) *Server {
	return &Server{
		rootFS: rootFS,
	}
}

// FirmwareFileInfo represents information about a firmware file.
type FirmwareFileInfo struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	IsDirectory bool      `json:"is_directory"`
	Permissions string    `json:"permissions"`
}

// ListFirmwareFiles lists all firmware files in the specified directory.
// This is a custom method that extends the standard gNOI File service
// to provide firmware-specific file listing capabilities.
func (s *Server) ListFirmwareFiles(ctx context.Context, directory string) ([]FirmwareFileInfo, error) {
	glog.V(2).Infof("gNOI File: ListFirmwareFiles called for directory: %s", directory)

	// Resolve the firmware directory path with rootFS
	resolvedPath := s.resolveFilesystemPath(directory)

	// Check if directory exists
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", directory)
	} else if err != nil {
		return nil, fmt.Errorf("failed to access directory %s: %v", directory, err)
	}

	var files []FirmwareFileInfo

	// Walk through the directory and collect file information
	err := filepath.WalkDir(resolvedPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			glog.Warningf("Error accessing path %s: %v", path, err)
			return nil // Continue processing other files
		}

		// Skip the root directory itself
		if path == resolvedPath {
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			glog.Warningf("Error getting info for %s: %v", path, err)
			return nil // Continue processing other files
		}

		// Get relative path from the firmware directory
		relPath, err := filepath.Rel(resolvedPath, path)
		if err != nil {
			glog.Warningf("Error getting relative path for %s: %v", path, err)
			relPath = filepath.Base(path)
		}

		// Create file info entry
		fileInfo := FirmwareFileInfo{
			Name:        relPath,
			Size:        info.Size(),
			ModTime:     info.ModTime(),
			IsDirectory: info.IsDir(),
			Permissions: info.Mode().String(),
		}

		files = append(files, fileInfo)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %v", directory, err)
	}

	glog.V(2).Infof("gNOI File: Found %d files in directory %s", len(files), directory)
	return files, nil
}

// GetFirmwareFileCount returns the count of files in a firmware directory.
func (s *Server) GetFirmwareFileCount(ctx context.Context, directory string) (int, error) {
	files, err := s.ListFirmwareFiles(ctx, directory)
	if err != nil {
		return 0, err
	}
	return len(files), nil
}

// GetFirmwareFileInfo returns information about a specific firmware file.
func (s *Server) GetFirmwareFileInfo(ctx context.Context, directory string, filename string) (
	*FirmwareFileInfo, error) {
	files, err := s.ListFirmwareFiles(ctx, directory)
	if err != nil {
		return nil, err
	}

	// Look for the specific file
	for _, file := range files {
		if file.Name == filename {
			return &file, nil
		}
	}

	return nil, fmt.Errorf("file not found: %s", filename)
}

// Standard gNOI File service methods (currently unimplemented)

// Get retrieves a file from the target device.
func (s *Server) Get(req *file.GetRequest, stream file.File_GetServer) error {
	glog.V(2).Info("gNOI File: Get RPC called")
	return status.Error(codes.Unimplemented, "Get RPC not yet implemented")
}

// Put transfers a file to the target device.
func (s *Server) Put(stream file.File_PutServer) error {
	glog.V(2).Info("gNOI File: Put RPC called")
	return status.Error(codes.Unimplemented, "Put RPC not yet implemented")
}

// Stat returns metadata about a file.
func (s *Server) Stat(ctx context.Context, req *file.StatRequest) (*file.StatResponse, error) {
	glog.V(2).Infof("gNOI File: Stat RPC called for path: %s", req.GetPath())
	return nil, status.Error(codes.Unimplemented, "Stat RPC not yet implemented")
}

// Remove deletes a file from the target device.
func (s *Server) Remove(ctx context.Context, req *file.RemoveRequest) (*file.RemoveResponse, error) {
	glog.V(2).Infof("gNOI File: Remove RPC called for file: %s", req.GetRemoteFile())
	return nil, status.Error(codes.Unimplemented, "Remove RPC not yet implemented")
}

// TransferToRemote transfers a file from the target to a remote location.
func (s *Server) TransferToRemote(ctx context.Context, req *file.TransferToRemoteRequest) (
	*file.TransferToRemoteResponse, error) {
	glog.V(2).Infof("gNOI File: TransferToRemote RPC called")
	return nil, status.Error(codes.Unimplemented, "TransferToRemote RPC not yet implemented")
}

// resolveFilesystemPath resolves a filesystem path with the server's rootFS.
// This handles the difference between containerized and bare-metal deployments.
func (s *Server) resolveFilesystemPath(fsPath string) string {
	// If no rootFS is configured or it's root, use the path as-is
	if s.rootFS == "" || s.rootFS == "/" {
		return fsPath
	}

	// If the path is already absolute and starts with rootFS, use as-is
	if strings.HasPrefix(fsPath, s.rootFS) {
		return fsPath
	}

	// For containerized deployments, resolve the path within rootFS
	if strings.HasPrefix(fsPath, "/") {
		// Absolute path - join with rootFS
		return filepath.Join(s.rootFS, fsPath)
	}

	// Relative path - use as-is (though this is unusual for filesystem queries)
	return fsPath
}
