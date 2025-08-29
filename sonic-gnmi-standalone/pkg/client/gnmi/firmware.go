package gnmi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// FirmwareFileInfo represents information about a firmware file.
type FirmwareFileInfo struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	IsDirectory bool      `json:"is_directory"`
	Permissions string    `json:"permissions"`
}

// FirmwareFilesResponse represents the response for firmware files listing.
type FirmwareFilesResponse struct {
	Directory string             `json:"directory"`
	FileCount int                `json:"file_count"`
	Files     []FirmwareFileInfo `json:"files"`
}

// GetFirmwareFiles retrieves a list of firmware files in the specified directory.
// This is a convenience method that constructs the appropriate gNMI path and
// handles the response parsing.
func (c *Client) GetFirmwareFiles(ctx context.Context, directory string) (*FirmwareFilesResponse, error) {
	if directory == "" {
		return nil, fmt.Errorf("firmware directory is required")
	}

	// Construct the gNMI path: /sonic/system/firmware[directory=<directory>]/files
	path := &gnmi.Path{
		Elem: []*gnmi.PathElem{
			{Name: "sonic"},
			{Name: "system"},
			{
				Name: "firmware",
				Key:  map[string]string{"directory": directory},
			},
			{Name: "files"},
		},
	}

	glog.V(2).Infof("Requesting firmware files for directory: %s", directory)

	// Make the gNMI Get request
	resp, err := c.Get(ctx, []*gnmi.Path{path}, gnmi.Encoding_JSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get firmware files for directory %s: %w", directory, err)
	}

	// Parse the response
	if len(resp.Notification) == 0 || len(resp.Notification[0].Update) == 0 {
		return nil, fmt.Errorf("no data received for directory %s", directory)
	}

	update := resp.Notification[0].Update[0]
	jsonVal := update.Val.GetJsonVal()
	if jsonVal == nil {
		return nil, fmt.Errorf("expected JSON response, got %T", update.Val.Value)
	}

	// Unmarshal the JSON response
	var firmwareResp FirmwareFilesResponse
	if err := json.Unmarshal(jsonVal, &firmwareResp); err != nil {
		return nil, fmt.Errorf("failed to parse firmware files response: %w", err)
	}

	glog.V(2).Infof("Retrieved %d firmware files from directory %s", 
		firmwareResp.FileCount, directory)

	return &firmwareResp, nil
}

// GetFirmwareFileCount retrieves only the count of firmware files in the specified directory.
func (c *Client) GetFirmwareFileCount(ctx context.Context, directory string) (int, error) {
	if directory == "" {
		return 0, fmt.Errorf("firmware directory is required")
	}

	// Construct the gNMI path: /sonic/system/firmware[directory=<directory>]/files/count
	path := &gnmi.Path{
		Elem: []*gnmi.PathElem{
			{Name: "sonic"},
			{Name: "system"},
			{
				Name: "firmware",
				Key:  map[string]string{"directory": directory},
			},
			{Name: "files"},
			{Name: "count"},
		},
	}

	glog.V(2).Infof("Requesting firmware file count for directory: %s", directory)

	// Make the gNMI Get request
	resp, err := c.Get(ctx, []*gnmi.Path{path}, gnmi.Encoding_JSON)
	if err != nil {
		return 0, fmt.Errorf("failed to get firmware file count for directory %s: %w", directory, err)
	}

	// Parse the response
	if len(resp.Notification) == 0 || len(resp.Notification[0].Update) == 0 {
		return 0, fmt.Errorf("no data received for directory %s", directory)
	}

	update := resp.Notification[0].Update[0]
	jsonVal := update.Val.GetJsonVal()
	if jsonVal == nil {
		return 0, fmt.Errorf("expected JSON response, got %T", update.Val.Value)
	}

	// Unmarshal the JSON response (should be a simple number)
	var count int
	if err := json.Unmarshal(jsonVal, &count); err != nil {
		return 0, fmt.Errorf("failed to parse firmware file count response: %w", err)
	}

	return count, nil
}

// FirmwareFileResponse represents the response for a specific firmware file.
type FirmwareFileResponse struct {
	Directory string           `json:"directory"`
	File      FirmwareFileInfo `json:"file"`
}

// GetFirmwareFileInfo retrieves information about a specific firmware file.
func (c *Client) GetFirmwareFileInfo(ctx context.Context, directory string, filename string) (*FirmwareFileResponse, error) {
	if directory == "" {
		return nil, fmt.Errorf("firmware directory is required")
	}
	if filename == "" {
		return nil, fmt.Errorf("firmware filename is required")
	}

	// Construct the gNMI path: /sonic/system/firmware[directory=<directory>]/files/<filename>
	path := &gnmi.Path{
		Elem: []*gnmi.PathElem{
			{Name: "sonic"},
			{Name: "system"},
			{
				Name: "firmware",
				Key:  map[string]string{"directory": directory},
			},
			{Name: "files"},
			{Name: filename},
		},
	}

	glog.V(2).Infof("Requesting firmware file info for %s in directory: %s", filename, directory)

	// Make the gNMI Get request
	resp, err := c.Get(ctx, []*gnmi.Path{path}, gnmi.Encoding_JSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get firmware file info for %s in directory %s: %w", filename, directory, err)
	}

	// Parse the response
	if len(resp.Notification) == 0 || len(resp.Notification[0].Update) == 0 {
		return nil, fmt.Errorf("no data received for file %s in directory %s", filename, directory)
	}

	update := resp.Notification[0].Update[0]
	jsonVal := update.Val.GetJsonVal()
	if jsonVal == nil {
		return nil, fmt.Errorf("expected JSON response, got %T", update.Val.Value)
	}

	// Unmarshal the JSON response
	var fileResp FirmwareFileResponse
	if err := json.Unmarshal(jsonVal, &fileResp); err != nil {
		return nil, fmt.Errorf("failed to parse firmware file info response: %w", err)
	}

	glog.V(2).Infof("Retrieved info for firmware file %s: %d bytes, modified %s", 
		filename, fileResp.File.Size, fileResp.File.ModTime)

	return &fileResp, nil
}
