# Nessus Scan Merger Progress Bar Enhancement

## Overview

This document outlines the design for adding a visual progress bar to the existing Nessus scan merger CLI tool. The enhancement will provide real-time feedback to users during the file processing and merging operations, improving user experience by showing progress, estimated time remaining, and current operation status.

## Architecture

### Current Architecture Analysis

The existing application follows a simple CLI architecture:

```mermaid
flowchart TD
    A[Main Function] --> B[Find Nessus Files]
    B --> C[Process Files Sequentially]
    C --> D[Parse XML File]
    D --> E[Extract Hosts]
    E --> F[Merge Data]
    F --> G[Save Output File]
    
    C --> D1[File 1]
    C --> D2[File 2]
    C --> D3[File N]
```

### Enhanced Architecture with Progress Bar

```mermaid
flowchart TD
    A[Main Function] --> B[Initialize Progress Bar]
    B --> C[Find Nessus Files]
    C --> D[Setup Progress Tracking]
    D --> E[Process Files with Progress]
    E --> F[Update Progress Bar]
    F --> G[Parse XML File]
    G --> H[Extract Hosts]
    H --> I[Update Progress]
    I --> J[Merge Data]
    J --> K[Finalize Progress]
    K --> L[Save Output File]
    
    subgraph "Progress Operations"
        F --> F1[File Progress]
        I --> I1[Host Count Progress]
        K --> K1[Completion Status]
    end
```

## Progress Bar Implementation Strategy

### Technology Selection

**Selected Library:** `github.com/schollz/progressbar/v3`

**Rationale:**
- Lightweight and actively maintained
- Supports multiple progress indicators
- Customizable appearance and behavior
- Built-in support for bytes, iterations, and time estimation
- Compatible with Go 1.21

### Progress Tracking Strategy

**Single Progress Bar Approach**: Track total host processing progress across all files
- Calculate total host count from all .nessus files at the beginning
- Update progress bar as hosts are processed and merged
- Show current processed hosts vs total hosts with percentage

### Progress Bar Components

```mermaid
graph TD
    A[Single Progress Bar] --> B[Total Host Count]
    A --> C[Processed Hosts]
    A --> D[Current File Info]
    
    B --> B1[Pre-scan all files for host count]
    C --> C1[Hosts processed so far]
    C --> C2[Unique hosts added]
    C --> C3[Duplicate hosts skipped]
    
    D --> D1[Current file being processed]
    D --> D2[Processing speed (hosts/sec)]
```

## Implementation Design

### Data Structures Enhancement

```go
type ProgressTracker struct {
    ProgressBar     *progressbar.ProgressBar
    TotalHosts      int
    ProcessedHosts  int
    UniqueHosts     int
    DuplicateHosts  int
    CurrentFileName string
    StartTime       time.Time
}
```

### Function Modifications

#### Enhanced Main Function Flow

```mermaid
sequenceDiagram
    participant M as Main
    participant PT as ProgressTracker
    participant PB as ProgressBar
    participant FF as FindFiles
    participant SC as ScanCounter
    participant MF as MergeFiles
    
    M->>FF: Find Nessus Files
    FF-->>M: Return file list
    M->>SC: Pre-scan all files for total host count
    SC-->>M: Return total host count
    M->>PT: Initialize with total host count
    M->>PB: Create progress bar (0/totalHosts)
    
    loop For each file
        M->>PT: Set current file name
        M->>MF: Process file
        MF-->>M: Return parsed hosts
        loop For each host in file
            M->>PT: Process host (unique/duplicate)
            M->>PB: Update progress (+1 processed)
        end
    end
    
    M->>PB: Complete (100%)
    M->>PT: Save merged file
```

#### Progress Bar Display Format

**Single Unified Progress Bar:**

| Phase | Display Format |
|-------|----------------|
| Counting Hosts | `Scanning files for host count...` |
| Processing Hosts | `Processing hosts [████████░░] 800/1000 (80%) \| scan_file.nessus \| 45 hosts/sec` |
| Completed | `✓ Processed 1000 hosts (850 unique, 150 duplicates) in 22.3s` |

**Progress Information:**
- Current/Total hosts with percentage
- Current file being processed
- Processing speed (hosts per second)
- Unique vs duplicate host counts

### Error Handling with Progress

```mermaid
flowchart TD
    A[Progress Operation] --> B{Error Occurred?}
    B -->|No| C[Continue Progress]
    B -->|Yes| D[Pause Progress Bar]
    D --> E[Display Error Message]
    E --> F[Resume or Terminate]
    F -->|Resume| C
    F -->|Terminate| G[Cleanup Progress]
```

### Configuration Options

Progress bar behavior will be configurable through command-line flags:

| Flag | Description | Default |
|------|-------------|---------|
| `--quiet` | Disable progress bar | false |
| `--progress-style` | Progress bar style (bar, spinner, simple) | bar |
| `--show-speed` | Show processing speed | true |
| `--show-eta` | Show estimated time remaining | true |

## User Interface Design

### Progress Bar Visual Elements

```
Host Processing Progress:
┌─────────────────────────────────────────────────────────────┐
│ Processing hosts: scan_results_2024.nessus                 │
│ ████████████████████████░░░░░░░░░ 800/1000 (80%) | ETA: 4s │
│ Speed: 45 hosts/sec | Unique: 680 | Duplicates: 120       │
└─────────────────────────────────────────────────────────────┘

Completion Summary:
┌─────────────────────────────────────────────────────────────┐
│ ✓ Processed 1000 hosts from 7 files                       │
│ ✓ Unique hosts: 850 | Duplicates skipped: 150             │
│ ✓ Output saved: merged_nessus_report.nessus (2.8MB)       │
│ ✓ Completed in 22.3s (avg: 45 hosts/sec)                  │
└─────────────────────────────────────────────────────────────┘
```

## Testing Strategy

### Unit Testing Scope

1. **Progress Tracker Operations**
   - Progress calculation accuracy
   - State transitions
   - Error handling during progress updates

2. **Progress Bar Integration**
   - Progress bar initialization
   - Update mechanisms
   - Cleanup operations

3. **User Interface Testing**
   - Visual progress representation
   - Progress bar completion states
   - Error message display

### Integration Testing

```mermaid
graph TD
    A[Integration Test Suite] --> B[Small File Set Test]
    A --> C[Large File Set Test]
    A --> D[Error Scenario Test]
    A --> E[Performance Impact Test]
    
    B --> B1[2-3 small .nessus files]
    B --> B2[Verify progress accuracy]
    
    C --> C1[10+ large .nessus files]
    C --> C2[Test ETA calculation]
    
    D --> D1[Corrupted files]
    D --> D2[Permission errors]
    D --> D3[Progress bar recovery]
    
    E --> E1[Measure performance overhead]
    E --> E2[Memory usage impact]
```

## Performance Considerations

### Progress Bar Overhead

- Progress updates limited to reasonable intervals (every 100ms minimum)
- Efficient progress calculation to avoid performance degradation
- Memory-conscious progress tracking to handle large file sets

### Optimization Strategies

1. **Batched Updates**: Group multiple host additions before updating progress
2. **Conditional Rendering**: Skip progress updates for very fast operations
3. **Memory Efficiency**: Use lightweight progress tracking structures

## Dependencies Management

### New Dependencies

```go
// go.mod additions
require (
    github.com/schollz/progressbar/v3 v3.14.1
)
```

### Dependency Justification

- **progressbar/v3**: Mature, well-maintained library with comprehensive features
- **Minimal footprint**: Adds ~200KB to binary size
- **Zero breaking changes**: Maintains backward compatibility with existing functionality