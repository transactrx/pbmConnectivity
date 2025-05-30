# pbmConnectivity

A Go library for connecting to Pharmacy Benefit Manager (PBM) systems using various transport protocols including TLS, HTTP/HTTPS, and socket connections.

## Overview

pbmConnectivity provides a unified interface for communicating with PBM systems through multiple connectivity options:

- **TLS Synchronous**: Direct TLS connections for real-time messaging
- **TLS Persisted Synchronous**: Persistent TLS connections with connection reuse
- **HTTP/HTTPS**: RESTful API connections for modern web-based PBM systems
- **Socket Connections**: Both persistent and non-persistent socket-based communication
- **Asynchronous Flow**: Non-blocking message processing with queuing

## Features

- Multiple transport protocols (TLS, HTTP/HTTPS, TCP sockets)
- Configurable connection parameters and timeouts
- Header validation and message format checking
- Connection pooling and persistence
- Statistics collection and monitoring
- OAuth2 token management for HTTP connections
- FastHTTP support for high-performance HTTP operations

## Requirements

**Go 1.24.3 or later is required** due to critical security vulnerabilities in earlier versions.

## Installation

```bash
go get github.com/transactrx/pbmConnectivity
```

## Dependencies

- `github.com/transactrx/ncpdpDestination` - NCPDP message handling
- `github.com/valyala/fasthttp` - High-performance HTTP client
- `golang.org/x/oauth2` - OAuth2 authentication

## Usage

### Basic TLS Connection Example

```go
package main

import (
    "github.com/transactrx/pbmConnectivity/pkg/global"
    "github.com/transactrx/pbmConnectivity/pkg/tlspersistedsynch"
)

func main() {
    var tlsCon global.PBMConnectWithStats = &tlspersistedsynch.TLSPersistedSyncConnect{}
    
    config := map[string]interface{}{
        "pbmUrl":                  "10.0.120.250",
        "pbmPort":                 "30004",
        "pbmReceiveTimeOut":       "10",  
        "pbmInsecureSkipVerify":   true,
        "headerCheck":             true,
        "endOfRecordChar":         "LEN",
    }
    
    tlsCon.Start(config)
    
    header := map[string][]string{
        "transmissionId": {"123456789"},
    }
    
    claim := "M00001004261004336D0B1B..." // Your NCPDP message
    response, _, err := tlsCon.Post([]byte(claim), header)
    
    if err == nil {
        fmt.Printf("Response: %s", response)
    }
}
```

### HTTP/HTTPS Connection Example

```go
package main

import (
    "github.com/transactrx/pbmConnectivity/pkg/global"
    "github.com/transactrx/pbmConnectivity/pkg/https"
)

func main() {
    routeInfo := https.RouteInfo{
        RouteCode: "301",
        PbmUrl:    "https://messaging2.qs1.com/erx/v2017071/eMar",
        Timeout:   5,
    }
    
    tlsSync := https.HTTPPBMConnect{Conf: routeInfo}
    var tlsCon global.PBMConnectWithStats = &tlsSync
    
    config := map[string]interface{}{
        "pbmUrl":                "https://messaging2.qs1.com/erx/v2017071/eMar",
        "pbmInsecureSkipVerify": true,
        "debugEnabled":          true,
    }
    
    tlsCon.Start(config)
    
    header := map[string][]string{
        "Content-Type": {"text/xml"},
    }
    
    response, _, err := tlsCon.Post([]byte("your_message"), header)
}
```

## Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `pbmUrl` | PBM server URL or IP address | Required |
| `pbmPort` | PBM server port | Required |
| `pbmReceiveTimeOut` | Receive timeout in seconds | "10" |
| `pbmQueueTimeOut` | Queue timeout in seconds | "10" |
| `pbmInsecureSkipVerify` | Skip TLS certificate verification | false |
| `pbmOutboundChnls` | Number of outbound channels | "1" |
| `headerCheck` | Enable header validation | false |
| `endOfRecordChar` | End of record character ("ETX", "LEN") | "ETX" |
| `debugEnabled` | Enable debug logging | false |

## Available Packages

### Core Interface
- `pkg/global` - Common interfaces (`PBMConnect`, `PBMConnectWithStats`)

### Transport Implementations
- `pkg/tlssynch` - Synchronous TLS connections
- `pkg/tlspersistedsynch` - Persistent synchronous TLS connections
- `pkg/socketsynch` - Synchronous socket connections  
- `pkg/socketpersistedsynch` - Persistent synchronous socket connections
- `pkg/https` - HTTP/HTTPS connections with FastHTTP
- `pkg/asynchflow` - Asynchronous message processing

### Utilities
- `pkg/helpers` - Common utility functions

## Building and Running

### Run Tests:
```bash
# Run all unit tests
make test

# Run tests with coverage report
make test-coverage

# Run CI pipeline (test + build)
make ci
```

### Build the library:
```bash
make build
```

### Run examples:
```bash
# HTTP example
make run

# Or build specific examples
go build -o httpsample cmd/httpPBM/main.go
go build -o asynchsample cmd/asynchflowpbm/main.go
go build -o tlssample cmd/examplePBM/main.go
```

## Testing

This project includes comprehensive unit tests covering:

- **Configuration Management**: All helper utilities with edge case testing
- **Interface Contracts**: Mock implementations and error handling
- **OAuth2 Authentication**: Token lifecycle, refresh, and validation
- **Connection Management**: Load balancing, site health, and failover
- **Concurrency**: Thread-safe operations and atomic counters

### Test Coverage:
- `pkg/helpers`: 100% coverage of utility functions
- `pkg/global`: Interface contract validation  
- `pkg/https`: OAuth2 token management and HTTP status mapping
- `pkg/asynchflow`: Concurrent TLS session management
- `pkg/tlssynch`: Site health monitoring and load balancing
- `pkg/socketsynch`: Socket connection management

### CI/CD Integration:
The GitHub Actions workflows automatically:
- Run tests on Go 1.24.3 (minimum required version)
- Generate coverage reports
- Build example applications
- Upload artifacts and coverage to Codecov
- Ensure all tests pass before releasing

## Examples

The `cmd/` directory contains complete working examples:

- `cmd/httpPBM/main.go` - HTTP/HTTPS connectivity example
- `cmd/asynchflowpbm/main.go` - Asynchronous flow example  
- `cmd/examplePBM/main.go` - TLS persisted synchronous example

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is part of the TransactRx ecosystem for pharmaceutical transaction processing.