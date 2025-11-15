# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go wrapper for the Microsoft SimConnect SDK, using CGO to call SimConnect API functions. The project enables Go applications to interface with Microsoft Flight Simulator through the SimConnect.dll.

**Chinese description from README**: SIMCONNECT SDK 的 Go 封装，使用 cgo 方式调用 SIMCONNECT SDK 的相关接口

## Architecture

### Core Components

- **simconnect/simconnect.go**: Main wrapper implementation
  - `SimConnect` struct manages the DLL handle and connection handle
  - `LoadDLL()`: Loads SimConnect.dll from the `simconnect/` directory
  - `Open()`: Establishes connection to the simulator with a client name
  - `Close()`: Closes the SimConnect session
  - All API calls use `syscall.DLL.FindProc()` to dynamically load functions from the DLL
  - Functions return HRESULT error codes (0 = success, non-zero = error with format 0x%x)

- **simconnect/types.go**: Constants and type definitions
  - `HRESULT` type for Windows API return values
  - Period constants for data request intervals (ONCE, VISUAL_FRAME, SECOND)
  - Object ID constants (SIMCONNECT_UNUSED)

- **simconnect/callbacks.go**: Callback infrastructure (incomplete)
  - `DispatchProc` callback type for handling SimConnect messages
  - `SetDispatchCallback()` stub - requires goroutine or Windows message loop implementation

- **main.go**: Example usage demonstrating the connection flow
  - Standard pattern: LoadDLL → Open → AddToDataDefinition → RequestDataOnSimObject → Close

### CGO/Syscall Architecture

The wrapper uses Go's `syscall` package (not traditional CGO) to directly call Windows DLL functions:

1. DLL is loaded via `syscall.LoadDLL()` with path `simconnect/SimConnect.dll`
2. Functions are resolved dynamically at runtime using `FindProc()`
3. All calls use `proc.Call()` with `uintptr` parameters and `unsafe.Pointer` conversions
4. String parameters are converted to UTF-16 via `syscall.StringToUTF16Ptr()`

### Data Flow

1. Client creates SimConnect instance via `LoadDLL()`
2. Opens connection with `Open(clientName)`
3. Defines data structures with `AddToDataDefinition()` (datum name, units, type)
4. Requests data from sim objects with `RequestDataOnSimObject()` (request ID, definition ID, object ID, period)
5. Data would be received via callbacks (not yet fully implemented)

## Building and Running

### Prerequisites
- Windows environment (SimConnect.dll is Windows-only)
- Go 1.24.7 or later
- SimConnect.dll must be present in `simconnect/` directory
- Microsoft Flight Simulator running for testing

### Build
```bash
go build -o simconnect_app.exe main.go
```

### Run
```bash
go run main.go
```

Expected output: "SimConnect opened successfully!" if Flight Simulator is running

### Module Path
Module name: `github.com/NovaVoyager/Microsoft_Simconnect`

Import the simconnect package:
```go
import "github.com/NovaVoyager/Microsoft_Simconnect/simconnect"
```

## Development Notes

### Adding New SimConnect API Functions

When wrapping additional SimConnect SDK functions:

1. Add the function as a method on `*SimConnect` in `simconnect/simconnect.go`
2. Use `s.Dll.FindProc("SimConnect_FunctionName")` to get the proc
3. Convert Go parameters to appropriate `uintptr` types
4. Use `unsafe.Pointer` for pointer conversions
5. Call via `proc.Call(s.handle, param1, param2, ...)`
6. Check return value for HRESULT errors (non-zero = error)
7. Add any new constants to `types.go`

### Error Handling Pattern

All wrapped functions follow this pattern:
```go
proc, err := s.Dll.FindProc("SimConnect_FunctionName")
if err != nil {
    return err
}

r, _, _ := proc.Call(s.handle, ...)
if r != 0 {
    return fmt.Errorf("SimConnect_FunctionName failed, HRESULT=0x%x", r)
}
return nil
```

### Memory Management

- The `SimConnect` struct must call `Dll.Release()` when done (use defer)
- Always defer `Close()` after successful `Open()`
- UTF-16 string pointers are managed by Go's GC after syscall completion

### Incomplete Features

- **Callbacks**: `SetDispatchCallback()` is stubbed. A complete implementation needs:
  - Continuous goroutine calling `SimConnect_CallDispatch()`
  - Callback function marshaling from Go to C
  - Message parsing and dispatch to user handlers

- **Message Processing**: No message receiving/dispatching infrastructure yet

- **Data Structure Definitions**: Only basic `AddToDataDefinition()` implemented; more complex data structures need additional wrapper functions
