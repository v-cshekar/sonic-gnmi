# Debugging SONIC Image Architecture

## Method 1: Using Delve Debugger

### Debug the Test
```bash
# Run the test with delve
dlv test ./tests/loopback -- -test.run TestGNMISonicImageLoopback/list_all_sonic_image_files

# Inside delve, set breakpoints:
(dlv) break /home/chandra/wipr/sonic-gnmi/sonic-gnmi-standalone/pkg/client/gnmi/sonic_image.go:53
(dlv) break /home/chandra/wipr/sonic-gnmi/sonic-gnmi-standalone/pkg/server/gnmi/get.go:67
(dlv) break /home/chandra/wipr/sonic-gnmi/sonic-gnmi-standalone/pkg/server/gnmi/filesystem.go:46
(dlv) break /home/chandra/wipr/sonic-gnmi/sonic-gnmi-standalone/internal/file/file.go:28

# Run and step through
(dlv) continue
(dlv) next    # Step to next line
(dlv) step    # Step into function
(dlv) print directory   # Print variable values
(dlv) locals  # Show all local variables
```

### Debug the Server Binary
```bash
# Build and debug the server
make build
dlv exec ./bin/sonic-gnmi-standalone

# Set breakpoints on server startup
(dlv) break /home/chandra/wipr/sonic-gnmi/sonic-gnmi-standalone/pkg/server/gnmi/get.go:67
(dlv) continue

# In another terminal, run a client request:
# grpcurl -plaintext -d '{"path":[{"elem":[{"name":"sonic"},{"name":"system"},{"name":"sonic-image","key":{"directory":"/tmp"}},{"name":"files"}]}]}' localhost:50055 gnmi.gNMI/Get
```

## Method 2: Enhanced Logging for Tracing

Add detailed logging at each step to trace the complete flow.
