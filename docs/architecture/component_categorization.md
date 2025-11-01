# Component Categorization: Core/Modules/Helper Structure

**Date:** 2025-11-01  
**Task:** Task 1.3 - Project Structure Assessment and Planning (Step 2)  
**Analyst:** Agent_Infrastructure_Foundation

---

## Executive Summary

This document categorizes all existing LUMA components into the proposed three-tier architecture:
- **core/**: Business logic - checker engine, worker management, task distribution
- **modules/**: Infrastructure concerns - parsing, proxy, configuration, workflow systems
- **helper/**: Utilities - logging, export, common utilities

Each categorization includes detailed rationale and identifies components that don't fit cleanly into categories.

---

## 1. Categorization Framework

### 1.1 Category Definitions

**CORE (Business Logic Layer)**
- **Purpose:** Application-specific business logic and orchestration
- **Characteristics:**
  - Domain-specific functionality
  - Orchestrates modules to achieve business goals
  - Contains application state and workflow coordination
  - Not reusable outside this application
- **Dependencies:** Can depend on modules and helpers

**MODULES (Infrastructure Layer)**
- **Purpose:** Self-contained, reusable subsystems
- **Characteristics:**
  - Domain-agnostic or loosely coupled to domain
  - Could be extracted as standalone libraries
  - Provide specific technical capabilities
  - Well-defined interfaces
- **Dependencies:** Can depend on helpers, should not depend on core

**HELPER (Utility Layer)**
- **Purpose:** Generic utilities and cross-cutting concerns
- **Characteristics:**
  - No business logic
  - Pure functions or simple services
  - Highly reusable
  - Minimal dependencies
- **Dependencies:** Should have no dependencies on core or modules

### 1.2 Decision Criteria

| Criterion | Core | Modules | Helper |
|-----------|------|---------|--------|
| Business Logic | High | Low | None |
| Reusability | Low | High | Very High |
| Domain Coupling | Tight | Loose | None |
| Complexity | High | Medium | Low |
| Testability | Needs Integration | Unit Testable | Pure Functions |

---

## 2. Component Categorization

### 2.1 CORE Components

#### **2.1.1 Checker Engine**

**Files:**
- `internal/checker/checker.go` (943 lines)
- `internal/checker/global_checker.go` (775 lines)

**Category:** **CORE**

**Rationale:**
- Contains application-specific business logic for account checking
- Orchestrates multiple modules (parsing, proxy, workflow, config)
- Manages application state (stats, worker pools, channels)
- Implements domain-specific patterns (combo checking, result aggregation)
- Not reusable outside account checking domain

**Responsibilities:**
- Worker pool management and task distribution
- Combo processing orchestration
- Result aggregation and statistics
- Integration of parsing, proxy, and config modules
- Application lifecycle management

**Dependencies:**
- Modules: parsing, proxy, workflow, config
- Helpers: logging, export, utils

**Proposed Location:** `core/engine/`

---

#### **2.1.2 Worker Management**

**Current State:** Embedded in checker.go and global_checker.go

**Category:** **CORE**

**Rationale:**
- Worker pool pattern is application-specific
- Task distribution logic tied to combo checking
- Channel-based communication specific to checker workflow
- State management for worker coordination

**Responsibilities:**
- Worker lifecycle (spawn, monitor, shutdown)
- Task channel management
- Result channel aggregation
- Worker synchronization (WaitGroup, Context)

**Proposed Location:** `core/worker/`

**Note:** Should be extracted from checker.go into separate component

---

#### **2.1.3 Task Distribution**

**Current State:** Embedded in checker engines

**Category:** **CORE**

**Rationale:**
- Combo-to-task conversion is domain-specific
- Task prioritization based on application logic
- Rate limiting tied to business requirements (CPM)

**Responsibilities:**
- Convert combos to worker tasks
- Implement rate limiting and throttling
- Task queue management
- Priority-based distribution

**Proposed Location:** `core/distributor/`

**Note:** Should be extracted and made explicit

---

#### **2.1.4 Result Aggregation**

**Current State:** Embedded in checker engines

**Category:** **CORE**

**Rationale:**
- Result processing is application-specific
- Statistics calculation tied to business metrics
- Status determination (valid/invalid/error) is domain logic

**Responsibilities:**
- Collect results from workers
- Update statistics (hits, fails, CPM, etc.)
- Trigger export operations
- Real-time status updates

**Proposed Location:** `core/aggregator/`

**Note:** Should be extracted for clarity

---

### 2.2 MODULES Components

#### **2.2.1 Parsing Module**

**Files:**
- `internal/checker/parsing_engine.go` (75 lines)
- `internal/checker/json_parser.go` (467 bytes)
- `internal/checker/css_parser.go` (555 bytes)
- `internal/checker/regex_parser.go` (433 bytes)
- `internal/checker/lr_parser.go` (529 bytes)

**Category:** **MODULES**

**Rationale:**
- Self-contained parsing subsystem
- Domain-agnostic (can parse any text/JSON/HTML)
- Well-defined interface (Parser interface)
- Reusable in other contexts (web scraping, data extraction)
- No business logic, pure technical capability

**Responsibilities:**
- Parse various formats (JSON, CSS, Regex, LR)
- Strategy pattern implementation
- Parser orchestration and dispatch
- Error handling for parsing operations

**Proposed Location:** `modules/parsing/`

**Interface:**
```go
type Parser interface {
    Parse(input string, params ...string) ([]string, error)
}
```

---

#### **2.2.2 Workflow Module**

**Files:**
- `internal/checker/workflow_engine.go` (135 lines)
- `internal/checker/function_block.go` (3.7KB)
- `internal/checker/variable_manager.go` (1.4KB)
- `internal/checker/variable_manipulator.go` (4.9KB)

**Category:** **MODULES**

**Rationale:**
- Self-contained workflow execution system
- Could be used for any multi-step data processing
- Loosely coupled to domain (uses generic variables)
- Reusable for ETL, data transformation pipelines
- Well-defined workflow structure

**Responsibilities:**
- Execute multi-step workflows
- Manage variable state across steps
- Apply transformations (functions)
- Coordinate parsing and function operations

**Proposed Location:** `modules/workflow/`

**Interface:**
```go
type Engine interface {
    Execute(workflow Workflow, input string) error
    GetVariable(name string) (*Variable, error)
    Reset()
}
```

---

#### **2.2.3 Proxy Module**

**Files:**
- `internal/checker/advanced_proxy_manager.go` (507 lines)
- `internal/checker/proxy_health_monitor.go` (~12KB)
- `internal/proxy/scraper.go` (11.6KB)

**Category:** **MODULES**

**Rationale:**
- Complete proxy management subsystem
- Reusable for any application needing proxy rotation
- Domain-agnostic (not specific to account checking)
- Self-contained with clear responsibilities
- Could be extracted as standalone library

**Responsibilities:**
- Proxy acquisition (scraping)
- Proxy selection strategies
- Health monitoring and validation
- Performance tracking and scoring
- Geographic information management

**Proposed Location:** `modules/proxy/`

**Interface:**
```go
type Manager interface {
    GetProxy() (*types.Proxy, error)
    ReturnProxy(proxy *types.Proxy, success bool, latency int)
}
```

---

#### **2.2.4 Configuration Module**

**Files:**
- `internal/config/parser.go` (682 lines)
- `internal/config/manager.go` (10.8KB)

**Category:** **MODULES**

**Rationale:**
- Self-contained configuration management
- Specific to config formats (.opk, .svb, .loli) but modular
- Could be reused in other OpenBullet-compatible tools
- Well-defined parsing and management responsibilities

**Responsibilities:**
- Parse multiple config formats
- Validate configurations
- Config lifecycle management

**Proposed Location:** `modules/config/`

**Interface:**
```go
type Parser interface {
    ParseConfig(filePath string) (*types.Config, error)
}
```

---

#### **2.2.5 HTTP Client Module**

**Files:**
- `pkg/httpclient/azuretls_client.go` (220 lines)

**Category:** **MODULES**

**Rationale:**
- Provides HTTP request capability with TLS fingerprinting
- Reusable for any HTTP-based application
- Domain-agnostic

**Responsibilities:**
- Execute HTTP requests with custom TLS fingerprints
- Proxy integration for requests
- JA3 fingerprinting support

**Proposed Location:** `modules/httpclient/`

---

#### **2.2.6 Reporting Module**

**Files:**
- `internal/reporting/report_generator.go` (2.3KB)

**Category:** **MODULES**

**Rationale:**
- Self-contained report generation
- Could be reused for different report types
- Provides specific technical capability

**Proposed Location:** `modules/reporting/`

---

### 2.3 HELPER Components

#### **2.3.1 Logging Helper**

**Files:**
- `internal/logger/structured_logger.go` (20.6KB)
- `internal/checker/logger.go` (1.8KB) - **LEGACY, TO BE REMOVED**

**Category:** **HELPER**

**Rationale:**
- Cross-cutting concern (used by all components)
- No business logic
- Pure infrastructure service

**Responsibilities:**
- Structured logging with JSON format
- Log level management
- Buffered output

**Proposed Location:** `helper/logger/`

**Action Required:** Remove legacy `internal/checker/logger.go`

---

#### **2.3.2 Export Helper**

**Files:**
- `internal/checker/exporter.go` (286 lines)

**Category:** **HELPER**

**Rationale:**
- Generic file export functionality
- No business logic (just formatting and writing)
- Simple utility service

**Responsibilities:**
- Export results in multiple formats (txt, json, csv)
- File organization and directory management

**Proposed Location:** `helper/export/`

---

#### **2.3.3 Utilities Helper**

**Files:**
- `pkg/utils/utils.go` (43 lines)
- `pkg/utils/correlation.go`

**Category:** **HELPER**

**Rationale:**
- Pure utility functions
- No state or business logic
- Highly reusable

**Responsibilities:**
- File operations
- Validation functions
- String sanitization

**Proposed Location:** `helper/utils/`

---

#### **2.3.4 Types Helper**

**Files:**
- `pkg/types/types.go` (260 lines)

**Category:** **HELPER**

**Rationale:**
- Shared data structures
- No logic, just type definitions
- Used across all layers

**Proposed Location:** `helper/types/`

---

## 3. Categorization Summary

### 3.1 Component Distribution

| Category | Components | Files | Current Location |
|----------|-----------|-------|------------------|
| **CORE** | 4 components | ~8 files | internal/checker/ |
| **MODULES** | 6 subsystems | ~20 files | internal/checker/, internal/*, pkg/ |
| **HELPER** | 4 utilities | ~10 files | internal/checker/, internal/logger/, pkg/ |
| **CMD** | 4 entry points | 4 files | cmd/ (unchanged) |

### 3.2 Proposed Structure

```
stuffer-go/
├── core/                     # Business Logic
│   ├── engine/              # Checker engines
│   ├── worker/              # Worker pool (extracted)
│   ├── distributor/         # Task distribution (extracted)
│   └── aggregator/          # Result aggregation (extracted)
│
├── modules/                  # Infrastructure
│   ├── parsing/             # Parsing subsystem
│   ├── workflow/            # Workflow engine
│   ├── proxy/               # Proxy management
│   ├── config/              # Configuration
│   ├── httpclient/          # HTTP client
│   └── reporting/           # Report generation
│
├── helper/                   # Utilities
│   ├── logger/              # Logging
│   ├── export/              # Export utilities
│   ├── utils/               # General utilities
│   └── types/               # Type definitions
│
└── cmd/                      # Entry points (unchanged)
    ├── cli/
    ├── global/
    ├── gui/
    └── test/
```

### 3.3 Dependency Flow

```
cmd/ → core/ → modules/ → helper/
       ↓         ↓
       └─────────┘
```

**Rules:**
- `core/` can depend on `modules/` and `helper/`
- `modules/` can depend on `helper/` only
- `helper/` has no internal dependencies
- `cmd/` can depend on any layer

---

## 4. Boundary Definitions

### 4.1 What Belongs in CORE

✅ **Include:**
- Checker engine orchestration
- Worker pool management
- Task distribution logic
- Result aggregation
- Application-specific workflows
- Business rules

❌ **Exclude:**
- Generic parsing logic
- Proxy management
- Configuration parsing
- Logging infrastructure
- File utilities

### 4.2 What Belongs in MODULES

✅ **Include:**
- Self-contained subsystems
- Reusable technical capabilities
- Domain-agnostic functionality
- Well-defined interfaces

❌ **Exclude:**
- Application orchestration
- Business logic
- Pure utilities

### 4.3 What Belongs in HELPER

✅ **Include:**
- Pure utility functions
- Cross-cutting concerns
- Generic operations
- Shared types

❌ **Exclude:**
- Business logic
- Complex state management

---

## 5. Rationale for Key Decisions

### 5.1 Why Workflow is MODULE (not CORE)

**Decision:** Place workflow engine in `modules/`

**Reasoning:**
- Generic workflow execution system
- No checker-specific logic
- Reusable for ETL, data pipelines, web scraping
- Well-defined structure makes it library-worthy

### 5.2 Why Export is HELPER (not MODULE)

**Decision:** Place export in `helper/`

**Reasoning:**
- Simple file writing utility
- No complex logic or state
- Pure formatting and I/O
- More utility than infrastructure

### 5.3 Why Proxy is MODULE (not CORE)

**Decision:** Place proxy management in `modules/`

**Reasoning:**
- Completely reusable for any proxy-needing application
- No domain-specific logic
- Self-contained with clear interface
- Already has library-quality design

---

## 6. Components Requiring Special Handling

### 6.1 Entry Points (cmd/)

**Status:** Keep in `cmd/` directory (unchanged)
- Standard Go project layout
- Entry points orchestrate but aren't part of core
- No categorization needed

### 6.2 Test Files

**Status:** Keep alongside implementation
- Unit tests: Same package as code
- Integration tests: Separate `tests/` directory
- Follow Go conventions

### 6.3 Legacy Code

**Action Required:**
- Remove `internal/checker/logger.go` after migration
- Ensure all components use `helper/logger/`

---

## 7. Migration Priorities

### 7.1 Low-Risk (Move First)

1. **Utilities** - Pure functions, minimal dependencies
2. **Types** - No logic, just definitions
3. **Individual parsers** - Self-contained
4. **Export helper** - Simple dependencies

### 7.2 Medium-Risk (Move Second)

1. **Logger** - Cross-cutting but well-defined
2. **Parsing engine** - Has dependents
3. **Workflow system** - Used by checker
4. **Reporting** - Simple module

### 7.3 High-Risk (Move Last)

1. **Proxy module** - Complex with monitor
2. **Config module** - Large and complex
3. **Checker engines** - Central to app
4. **HTTP client** - External dependencies

---

## 8. Interface Requirements

### 8.1 Core Interfaces Needed

```go
// core/engine/interface.go
type Engine interface {
    Start() error
    Stop() error
    GetStats() *types.CheckerStats
}

// core/worker/interface.go
type Pool interface {
    Start(ctx context.Context, count int)
    Submit(task types.WorkerTask)
    Results() <-chan types.WorkerResult
}
```

### 8.2 Module Interfaces Needed

```go
// modules/parsing/interface.go
type Parser interface {
    Parse(input string, params ...string) ([]string, error)
}

// modules/proxy/interface.go
type Manager interface {
    GetProxy() (*types.Proxy, error)
    ReturnProxy(proxy *types.Proxy, success bool, latency int)
}

// modules/workflow/interface.go
type Engine interface {
    Execute(workflow Workflow, input string) error
}
```

### 8.3 Helper Interfaces Needed

```go
// helper/logger/interface.go
type Logger interface {
    Info(msg string, fields map[string]interface{})
    Error(msg string, fields map[string]interface{})
}

// helper/export/interface.go
type Exporter interface {
    Export(data interface{}, path string) error
}
```

---

## 9. Benefits of This Categorization

### 9.1 Clear Separation of Concerns
- Business logic isolated in core
- Infrastructure modularized
- Utilities easily accessible

### 9.2 Improved Testability
- Core testable with mocked modules
- Modules unit testable independently
- Helpers are pure functions

### 9.3 Enhanced Reusability
- Modules extractable as libraries
- Helpers usable in other projects
- Core remains application-specific

### 9.4 Better Maintainability
- Clear boundaries reduce cognitive load
- Easy component location
- Obvious placement for new features

---

## 10. Conclusion

All 38 components categorized into three-tier architecture:

**CORE (4 components):**
- Checker engine, worker management, task distribution, result aggregation

**MODULES (6 subsystems):**
- Parsing, workflow, proxy, config, httpclient, reporting

**HELPER (4 utilities):**
- Logging, export, utilities, types

This categorization provides:
- ✅ Clear separation between business logic, infrastructure, and utilities
- ✅ Well-defined boundaries with explicit rationale
- ✅ Improved testability through interface-based design
- ✅ Enhanced reusability of modules and helpers
- ✅ Scalable architecture for future growth

---

**Status:** Step 2 Complete - Ready for Step 3 (Detailed Directory Design)

**Next Step:** Design detailed directory structure with feature-based grouping, file organization, and naming conventions.

---

**Categorization Completed:** 2025-11-01  
**Analyst:** Agent_Infrastructure_Foundation
