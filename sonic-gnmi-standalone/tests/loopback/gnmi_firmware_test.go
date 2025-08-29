package loopback

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGNMIFirmwareLoopback tests the complete client-server loopback
// for the gNMI firmware file listing functionality.
func TestGNMIFirmwareLoopback(t *testing.T) {
	// Setup test infrastructure
	tempDir := t.TempDir()
	
	// Important: Enable gNOI File service for firmware functionality
	testServer := SetupInsecureTestServer(t, tempDir, []string{"gnmi", "gnoi.file"})
	defer testServer.Stop()

	client := SetupGNMIClient(t, testServer.Addr, 10*time.Second)
	defer client.Close()

	// Create test firmware directory structure
	firmwareDir := filepath.Join(tempDir, "test-firmware")
	err := os.MkdirAll(firmwareDir, 0755)
	require.NoError(t, err, "Failed to create firmware directory")

	// Create test firmware files
	testFiles := map[string]string{
		"device1.bin":      "firmware content for device 1",
		"device2.fw":       "firmware content for device 2",
		"bootloader.img":   "bootloader firmware image",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(firmwareDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err, "Failed to create test file %s", filename)
	}

	// Create subdirectory with nested files
	subDir := filepath.Join(firmwareDir, "drivers")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err, "Failed to create subdirectory")

	err = os.WriteFile(filepath.Join(subDir, "driver1.bin"), []byte("driver firmware 1"), 0644)
	require.NoError(t, err, "Failed to create nested file")

	err = os.WriteFile(filepath.Join(subDir, "driver2.bin"), []byte("driver firmware 2"), 0644)
	require.NoError(t, err, "Failed to create nested file")

	// Test contexts
	ctx, cancel := WithTestTimeout(15 * time.Second)
	defer cancel()

	t.Run("capabilities_include_firmware", func(t *testing.T) {
		resp, err := client.Capabilities(ctx)
		require.NoError(t, err, "Capabilities RPC failed")
		require.NotNil(t, resp)

		// Check that firmware model is advertised
		modelNames := make([]string, len(resp.SupportedModels))
		for i, model := range resp.SupportedModels {
			modelNames[i] = model.Name
		}
		assert.Contains(t, modelNames, "sonic-firmware", "sonic-firmware model should be advertised")
	})

	t.Run("list_all_firmware_files", func(t *testing.T) {
		// Test listing all firmware files
		firmwareResp, err := client.GetFirmwareFiles(ctx, firmwareDir)
		require.NoError(t, err, "GetFirmwareFiles RPC failed")
		require.NotNil(t, firmwareResp)

		assert.Equal(t, firmwareDir, firmwareResp.Directory, "Directory mismatch")
		assert.Equal(t, 6, firmwareResp.FileCount, "Expected 6 items (3 files + 1 subdir + 2 nested files)")
		assert.Len(t, firmwareResp.Files, 6, "Files slice length mismatch")

		// Verify we have both files and directories
		var fileCount, dirCount int
		fileNames := make(map[string]bool)
		
		for _, file := range firmwareResp.Files {
			fileNames[file.Name] = true
			if file.IsDirectory {
				dirCount++
			} else {
				fileCount++
			}
			
			// Verify common properties
			assert.NotEmpty(t, file.Name, "File name should not be empty")
			assert.NotEmpty(t, file.Permissions, "Permissions should not be empty")
			assert.False(t, file.ModTime.IsZero(), "Modification time should be set")
			
			if !file.IsDirectory {
				assert.Greater(t, file.Size, int64(0), "File size should be positive for files")
			}
		}

		assert.Equal(t, 5, fileCount, "Expected 5 files (3 top-level + 2 nested)")
		assert.Equal(t, 1, dirCount, "Expected 1 directory")

		// Verify specific files exist
		expectedFiles := []string{"device1.bin", "device2.fw", "bootloader.img", "drivers", "drivers/driver1.bin", "drivers/driver2.bin"}
		for _, expectedFile := range expectedFiles {
			assert.True(t, fileNames[expectedFile], "Expected file %s not found", expectedFile)
		}
	})

	t.Run("get_firmware_file_count", func(t *testing.T) {
		// Test getting just the file count
		count, err := client.GetFirmwareFileCount(ctx, firmwareDir)
		require.NoError(t, err, "GetFirmwareFileCount RPC failed")

		assert.Equal(t, 6, count, "Expected 6 items total")
	})

	t.Run("get_specific_firmware_file_info", func(t *testing.T) {
		// Test getting info for a specific file
		fileResp, err := client.GetFirmwareFileInfo(ctx, firmwareDir, "device1.bin")
		require.NoError(t, err, "GetFirmwareFileInfo RPC failed")
		require.NotNil(t, fileResp)

		assert.Equal(t, firmwareDir, fileResp.Directory, "Directory mismatch")
		assert.Equal(t, "device1.bin", fileResp.File.Name, "Filename mismatch")
		assert.False(t, fileResp.File.IsDirectory, "Should be a file, not directory")
		assert.Equal(t, int64(len("firmware content for device 1")), fileResp.File.Size, "File size mismatch")
		assert.NotEmpty(t, fileResp.File.Permissions, "Permissions should not be empty")
	})

	t.Run("get_nested_file_info", func(t *testing.T) {
		// Test getting info for a nested file
		fileResp, err := client.GetFirmwareFileInfo(ctx, firmwareDir, "drivers/driver1.bin")
		require.NoError(t, err, "GetFirmwareFileInfo RPC failed for nested file")
		require.NotNil(t, fileResp)

		assert.Equal(t, firmwareDir, fileResp.Directory, "Directory mismatch")
		assert.Equal(t, "drivers/driver1.bin", fileResp.File.Name, "Nested filename mismatch")
		assert.False(t, fileResp.File.IsDirectory, "Should be a file, not directory")
		assert.Equal(t, int64(len("driver firmware 1")), fileResp.File.Size, "Nested file size mismatch")
	})

	t.Run("get_directory_info", func(t *testing.T) {
		// Test getting info for a directory
		fileResp, err := client.GetFirmwareFileInfo(ctx, firmwareDir, "drivers")
		require.NoError(t, err, "GetFirmwareFileInfo RPC failed for directory")
		require.NotNil(t, fileResp)

		assert.Equal(t, firmwareDir, fileResp.Directory, "Directory mismatch")
		assert.Equal(t, "drivers", fileResp.File.Name, "Directory name mismatch")
		assert.True(t, fileResp.File.IsDirectory, "Should be a directory")
		assert.Greater(t, fileResp.File.Size, int64(0), "Directory size should be positive")
	})

	t.Run("error_nonexistent_directory", func(t *testing.T) {
		// Test error case - non-existent directory
		nonExistentDir := filepath.Join(tempDir, "does-not-exist")
		
		_, err := client.GetFirmwareFiles(ctx, nonExistentDir)
		assert.Error(t, err, "Should fail for non-existent directory")
		assert.Contains(t, err.Error(), "directory does not exist", "Error should mention directory not found")
	})

	t.Run("error_nonexistent_file", func(t *testing.T) {
		// Test error case - non-existent file
		_, err := client.GetFirmwareFileInfo(ctx, firmwareDir, "nonexistent.bin")
		assert.Error(t, err, "Should fail for non-existent file")
		assert.Contains(t, err.Error(), "file not found", "Error should mention file not found")
	})

	t.Run("error_empty_directory_path", func(t *testing.T) {
		// Test error case - empty directory
		_, err := client.GetFirmwareFiles(ctx, "")
		assert.Error(t, err, "Should fail for empty directory")
		assert.Contains(t, err.Error(), "firmware directory is required", "Error should mention directory required")
	})

	t.Run("error_empty_filename", func(t *testing.T) {
		// Test error case - empty filename
		_, err := client.GetFirmwareFileInfo(ctx, firmwareDir, "")
		assert.Error(t, err, "Should fail for empty filename")
		assert.Contains(t, err.Error(), "firmware filename is required", "Error should mention filename required")
	})
}

// TestGNMIFirmwareWithoutGNOIFile tests that firmware functionality fails
// when gNOI File service is not enabled.
func TestGNMIFirmwareWithoutGNOIFile(t *testing.T) {
	// Setup test infrastructure WITHOUT gNOI File service
	tempDir := t.TempDir()
	testServer := SetupInsecureTestServer(t, tempDir, []string{"gnmi"}) // Only gNMI, no gnoi.file
	defer testServer.Stop()

	client := SetupGNMIClient(t, testServer.Addr, 10*time.Second)
	defer client.Close()

	ctx, cancel := WithTestTimeout(10 * time.Second)
	defer cancel()

	t.Run("firmware_fails_without_gnoi_file_service", func(t *testing.T) {
		// This should fail because gNOI File service is not enabled
		_, err := client.GetFirmwareFiles(ctx, "/tmp")
		assert.Error(t, err, "Should fail when gNOI File service is not enabled")
		assert.Contains(t, err.Error(), "gNOI File service to be enabled", 
			"Error should mention gNOI File service requirement")
	})
}
