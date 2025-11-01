# Migration Strategy: Core/Modules/Helper Reorganization

**Date:** 2025-11-01  
**Task:** Task 1.3 - Project Structure Assessment and Planning (Step 4)  
**Analyst:** Agent_Infrastructure_Foundation

---

## Executive Summary

This document provides a detailed, phased migration strategy for reorganizing the LUMA codebase into the core/modules/helper architecture. The strategy prioritizes risk mitigation, interface preservation, and minimal disruption to development workflow.

**Migration Timeline:** 3-4 weeks (phased approach)  
**Risk Level:** Medium (managed through phasing)  
**Rollback Capability:** Full rollback possible at each phase

---

## 1. Migration Principles

### 1.1 Core Principles

**Incremental Migration**
- Move components in small, testable batches
- Verify functionality after each phase
- Maintain working state at all times

**Interface Preservation**
- All public APIs remain unchanged during migration
- Use type aliases for backward compatibility
- Gradual deprecation of old paths

**Testing at Every Step**
- Run full test suite after each move
- Add migration-specific tests
- Verify integration points

**Rollback Readiness**
- Each phase is independently reversible
- Git branches for each phase
- Clear rollback procedures documented

### 1.2 Success Criteria

**Phase Completion Criteria:**
- ✅ All tests passing
- ✅ No import path errors
- ✅ Documentation updated
- ✅ Build successful
- ✅ Functionality verified

**Overall Success Criteria:**
- ✅ New structure fully implemented
- ✅ All components categorized correctly
- ✅ Zero functionality regressions
- ✅ Improved code organization metrics
- ✅ Team onboarded to new structure

---

## 2. Migration Phases

### Phase 0: Preparation (Days 1-2)

**Objective:** Set up infrastructure for safe migration

**Tasks:**
1. Create migration branch: `feature/structure-reorganization`
2. Set up rollback procedures
3. Create migration tracking document
4. Backup current state (Git tag: `pre-migration-backup`)
5. Create directory structure skeleton
6. Update `.gitignore` if needed
7. Document current import paths

**Deliverables:**
- Migration branch created
- Empty directory structure (`core/`, `modules/`, `helper/`)
- Rollback documentation
- Import path inventory

**Verification:**
- Branch builds successfully
- All tests pass on migration branch
- Team briefed on migration plan

**Risk:** Low  
**Rollback:** Delete branch, return to main

---

### Phase 1: Helper Layer Migration (Days 3-5)

**Objective:** Move helper packages (lowest risk, no dependents)

**Components to Move:**
1. **helper/types/** ← `pkg/types/types.go`
2. **helper/utils/** ← `pkg/utils/`
3. **helper/logger/** ← `internal/logger/`
4. **helper/export/** ← `internal/checker/exporter.go`

**Step-by-Step Process:**

**Step 1.1: Move Types (Day 3)**
```bash
# Create structure
mkdir -p helper/types

# Split types.go into domain-specific files
# - checker.go (CheckerConfig, CheckerStats, WorkerTask, WorkerResult)
# - proxy.go (Proxy, ProxyMetrics, ProxyLocation, ProxyQuality)
# - config.go (Config, ConfigType)
# - result.go (CheckResult, BotStatus)
# - common.go (Common types)

# Move and split
git mv pkg/types/types.go helper/types/
# Then split into multiple files

# Create type aliases in old location for compatibility
# pkg/types/compat.go (temporary)
```

**Step 1.2: Move Utils (Day 3)**
```bash
mkdir -p helper/utils

# Split utils.go
git mv pkg/utils/utils.go helper/utils/file.go
# Then split into: file.go, validation.go, string.go
git mv pkg/utils/correlation.go helper/utils/

# Update imports across codebase
find . -name "*.go" -exec sed -i 's|universal-checker/pkg/utils|universal-checker/helper/utils|g' {} \;
```

**Step 1.3: Move Logger (Day 4)**
```bash
mkdir -p helper/logger

git mv internal/logger/structured_logger.go helper/logger/logger.go
# Split into: logger.go, formatters.go, interface.go, types.go

# Update imports
find . -name "*.go" -exec sed -i 's|universal-checker/internal/logger|universal-checker/helper/logger|g' {} \;
```

**Step 1.4: Move Export (Day 5)**
```bash
mkdir -p helper/export

git mv internal/checker/exporter.go helper/export/exporter.go
# Split into: exporter.go, formatters.go, interface.go, types.go

# Update imports in checker files
```

**Step 1.5: Remove Legacy Logger (Day 5)**
```bash
# Delete old logger after verification
rm internal/checker/logger.go

# Update any remaining references
grep -r "internal/checker/logger" --include="*.go"
# Fix any found
```

**Verification:**
```bash
# Run tests
go test ./helper/...
go test ./...

# Build all entry points
go build ./cmd/cli
go build ./cmd/global
go build ./cmd/gui

# Verify no old imports remain
grep -r "pkg/types" --include="*.go"
grep -r "pkg/utils" --include="*.go"
grep -r "internal/logger" --include="*.go"
```

**Deliverables:**
- All helper packages in new location
- Old locations removed
- All tests passing
- Documentation updated

**Risk:** Low (helpers have many dependents but are pure utilities)  
**Rollback:** Revert commits, restore from backup tag

---

### Phase 2: Modules Layer Migration (Days 6-12)

**Objective:** Move self-contained modules

**Components to Move:**
1. **modules/parsing/** ← `internal/checker/parsing_*.go` (Day 6-7)
2. **modules/httpclient/** ← `pkg/httpclient/` (Day 7)
3. **modules/reporting/** ← `internal/reporting/` (Day 8)
4. **modules/workflow/** ← `internal/checker/workflow_*.go`, variables (Day 9-10)
5. **modules/config/** ← `internal/config/` (Day 10-11)
6. **modules/proxy/** ← `internal/checker/proxy_*.go` + `internal/proxy/` (Day 11-12)

**Step 2.1: Move Parsing Module (Days 6-7)**
```bash
mkdir -p modules/parsing

# Move files
git mv internal/checker/parsing_engine.go modules/parsing/engine.go
git mv internal/checker/json_parser.go modules/parsing/json.go
git mv internal/checker/css_parser.go modules/parsing/css.go
git mv internal/checker/regex_parser.go modules/parsing/regex.go
git mv internal/checker/lr_parser.go modules/parsing/lr.go

# Create interface.go and types.go
# Extract interfaces and types from engine.go

# Update package declaration in all files
sed -i 's/package checker/package parsing/g' modules/parsing/*.go

# Update imports
find . -name "*.go" -exec sed -i 's|internal/checker\.Parser|modules/parsing\.Parser|g' {} \;
find . -name "*.go" -exec sed -i 's|internal/checker\.ParseType|modules/parsing\.ParseType|g' {} \;
```

**Step 2.2: Move HTTP Client (Day 7)**
```bash
mkdir -p modules/httpclient

git mv pkg/httpclient/azuretls_client.go modules/httpclient/azuretls.go

# Create interface.go, types.go
# Add standard.go for fallback implementation

# Update imports
find . -name "*.go" -exec sed -i 's|pkg/httpclient|modules/httpclient|g' {} \;
```

**Step 2.3: Move Reporting (Day 8)**
```bash
mkdir -p modules/reporting

git mv internal/reporting/report_generator.go modules/reporting/generator.go

# Create interface.go, types.go, templates.go, formatters.go

# Update imports
find . -name "*.go" -exec sed -i 's|internal/reporting|modules/reporting|g' {} \;
```

**Step 2.4: Move Workflow Module (Days 9-10)**
```bash
mkdir -p modules/workflow

# Move workflow-related files
git mv internal/checker/workflow_engine.go modules/workflow/engine.go
git mv internal/checker/function_block.go modules/workflow/function.go
git mv internal/checker/variable_manager.go modules/workflow/variable.go
git mv internal/checker/variable_manipulator.go modules/workflow/manipulator.go

# Create interface.go and types.go
# Extract Workflow, WorkflowStep types

# Update package
sed -i 's/package checker/package workflow/g' modules/workflow/*.go

# Update imports
find . -name "*.go" -exec sed -i 's|internal/checker\.WorkflowEngine|modules/workflow\.Engine|g' {} \;
```

**Step 2.5: Move Config Module (Days 10-11)**
```bash
mkdir -p modules/config

git mv internal/config/parser.go modules/config/
git mv internal/config/manager.go modules/config/

# Split parser.go into format-specific files
# parser.go (main), opk.go, svb.go, loli.go

# Create interface.go, validator.go

# Update imports
find . -name "*.go" -exec sed -i 's|internal/config|modules/config|g' {} \;
```

**Step 2.6: Move Proxy Module (Days 11-12)**
```bash
mkdir -p modules/proxy

# Move from internal/checker/
git mv internal/checker/advanced_proxy_manager.go modules/proxy/manager.go
git mv internal/checker/proxy_health_monitor.go modules/proxy/monitor.go

# Move from internal/proxy/
git mv internal/proxy/scraper.go modules/proxy/

# Extract strategies to separate file
# Create interface.go, types.go

# Update package
sed -i 's/package checker/package proxy/g' modules/proxy/*.go

# Update imports
find . -name "*.go" -exec sed -i 's|internal/checker\.AdvancedProxyManager|modules/proxy\.Manager|g' {} \;
find . -name "*.go" -exec sed -i 's|internal/checker\.ProxyHealthMonitor|modules/proxy\.Monitor|g' {} \;
```

**Verification After Phase 2:**
```bash
# Test all modules
go test ./modules/...

# Build and test
go test ./...
go build ./cmd/...

# Verify no old module imports
grep -r "internal/checker/parsing" --include="*.go"
grep -r "internal/checker/workflow" --include="*.go"
grep -r "internal/checker/proxy" --include="*.go"
grep -r "pkg/httpclient" --include="*.go"
```

**Deliverables:**
- All 6 modules in new locations
- Interfaces defined
- Tests passing
- Old locations removed

**Risk:** Medium (modules have dependencies but well-defined)  
**Rollback:** Revert Phase 2 commits, keep Phase 1

---

### Phase 3: Core Layer Migration (Days 13-18)

**Objective:** Extract and organize core business logic

**Components to Create/Move:**
1. **core/engine/** ← Extract from `internal/checker/checker.go`, `global_checker.go` (Days 13-15)
2. **core/worker/** ← Extract worker pool logic from checker (Days 15-16)
3. **core/distributor/** ← Extract task distribution (Day 16-17)
4. **core/aggregator/** ← Extract result aggregation (Day 17-18)

**Step 3.1: Refactor Checker Engines (Days 13-15)**

This is the most complex phase as it involves extracting logic.

```bash
mkdir -p core/engine

# Step 1: Create base_checker.go with shared logic
# Extract common code from checker.go and global_checker.go

# Step 2: Move files
cp internal/checker/checker.go core/engine/checker.go
cp internal/checker/global_checker.go core/engine/global_checker.go

# Step 3: Create interface.go
# Define Engine interface

# Step 4: Refactor to use new module imports
# Update all imports in checker.go and global_checker.go:
# - modules/parsing
# - modules/workflow  
# - modules/proxy
# - modules/config
# - helper/logger
# - helper/export
# - helper/types

# Step 5: Remove extracted logic (worker, distributor, aggregator)
# Will be done in subsequent steps
```

**Step 3.2: Extract Worker Pool (Days 15-16)**
```bash
mkdir -p core/worker

# Create new files (extract from checker.go)
# core/worker/pool.go - Worker pool management
# core/worker/worker.go - Individual worker logic
# core/worker/interface.go - Pool interface
# core/worker/types.go - Task/Result types if not in helper/types

# Update core/engine/checker.go to use worker.Pool interface
# Replace embedded worker logic with:
#   workerPool worker.Pool
```

**Step 3.3: Extract Task Distributor (Day 16-17)**
```bash
mkdir -p core/distributor

# Create new files
# core/distributor/distributor.go - Main logic
# core/distributor/rate_limiter.go - CPM rate limiting
# core/distributor/queue.go - Task queue
# core/distributor/interface.go - Distributor interface

# Update core/engine/checker.go:
#   distributor distributor.Distributor
```

**Step 3.4: Extract Result Aggregator (Day 17-18)**
```bash
mkdir -p core/aggregator

# Create new files  
# core/aggregator/aggregator.go - Result collection
# core/aggregator/statistics.go - Stats calculation
# core/aggregator/interface.go - Aggregator interface
# core/aggregator/types.go - Stats types

# Update core/engine/checker.go:
#   aggregator aggregator.Aggregator
```

**Step 3.5: Update Checker Constructor**
```go
// core/engine/checker.go
func NewChecker(config *types.CheckerConfig) *Checker {
    return &Checker{
        config: config,
        workerPool: worker.NewPool(config.MaxWorkers),
        distributor: distributor.NewDistributor(config.CPM),
        aggregator: aggregator.NewAggregator(),
        workflowEngine: workflow.NewEngine(),
        proxyManager: proxy.NewManager(proxy.StrategyBestScore),
        logger: logger.NewStructuredLogger(loggerConfig),
    }
}
```

**Verification After Phase 3:**
```bash
# Test core packages
go test ./core/...

# Integration tests
go test ./tests/integration/...

# Build
go build ./cmd/...

# Verify old internal/checker is empty or removed
ls internal/checker/
```

**Deliverables:**
- Core layer fully implemented
- Checker engines refactored
- Worker pool extracted
- Distributor extracted
- Aggregator extracted
- All interfaces defined

**Risk:** High (core refactoring, complex dependencies)  
**Rollback:** Revert Phase 3, keep Phases 1-2

---

### Phase 4: Cleanup and Finalization (Days 19-21)

**Objective:** Remove old structure, finalize migration

**Tasks:**

**Day 19: Remove Old Directories**
```bash
# Verify nothing remains that's needed
ls internal/checker/
ls internal/config/
ls internal/proxy/
ls internal/logger/
ls pkg/types/
ls pkg/utils/
ls pkg/httpclient/

# Remove empty directories
rm -rf internal/checker/
rm -rf internal/config/
rm -rf internal/proxy/
rm -rf internal/logger/
rm -rf pkg/types/
rm -rf pkg/utils/
rm -rf pkg/httpclient/

# Keep internal/ for future use
# Keep pkg/ for future use

# Remove internal/reporting if moved
rm -rf internal/reporting/
```

**Day 20: Update Documentation**
```bash
# Update README.md with new structure
# Update ARCHITECTURE.md
# Update import examples
# Update developer guide
# Create migration notes for team
```

**Day 21: Final Verification**
```bash
# Run full test suite
go test ./... -v

# Build all entry points
go build ./cmd/cli
go build ./cmd/global
go build ./cmd/gui
go build ./cmd/test

# Run linter
golangci-lint run ./...

# Verify import consistency
# No references to old paths
grep -r "pkg/types" --include="*.go" || echo "Clean"
grep -r "internal/checker" --include="*.go" || echo "Clean"
grep -r "internal/config" --include="*.go" || echo "Clean"

# Run integration tests
go test ./tests/integration/... -v

# Manual smoke test of CLI
./cli --help
```

**Deliverables:**
- Old directories removed
- Documentation updated
- All tests passing
- Build artifacts verified
- Team trained on new structure

**Risk:** Low (cleanup phase)  
**Rollback:** Revert Phase 4, keep Phases 1-3

---

## 3. Rollback Procedures

### 3.1 Phase-Specific Rollback

**Phase 0 Rollback:**
```bash
# Simple - delete branch
git checkout main
git branch -D feature/structure-reorganization
```

**Phase 1 Rollback (Helper Layer):**
```bash
# Revert all Phase 1 commits
git log --oneline --grep="Phase 1"  # Find commit range
git revert <commit-range>

# Or hard reset if not merged
git reset --hard pre-migration-backup

# Verify
go test ./...
go build ./cmd/...
```

**Phase 2 Rollback (Modules Layer):**
```bash
# If Phase 1 is stable, revert only Phase 2
git log --oneline --grep="Phase 2"
git revert <phase-2-commits>

# Restore modules to old locations
git checkout <pre-phase-2-commit> -- internal/checker/parsing_*
git checkout <pre-phase-2-commit> -- internal/checker/workflow_*
git checkout <pre-phase-2-commit> -- internal/checker/proxy_*
git checkout <pre-phase-2-commit> -- internal/config/
git checkout <pre-phase-2-commit> -- internal/proxy/
git checkout <pre-phase-2-commit> -- pkg/httpclient/
git checkout <pre-phase-2-commit> -- internal/reporting/

# Remove new directories
rm -rf modules/

# Run tests
go test ./...
```

**Phase 3 Rollback (Core Layer):**
```bash
# Revert Phase 3 commits
git log --oneline --grep="Phase 3"
git revert <phase-3-commits>

# Restore old checker files
git checkout <pre-phase-3-commit> -- internal/checker/checker.go
git checkout <pre-phase-3-commit> -- internal/checker/global_checker.go

# Remove core directory
rm -rf core/

# Run tests
go test ./...
```

**Phase 4 Rollback (Cleanup):**
```bash
# Easy - just restore old directories from git history
git checkout <pre-phase-4-commit> -- internal/
git checkout <pre-phase-4-commit> -- pkg/

# Remove documentation changes if needed
git checkout <pre-phase-4-commit> -- docs/
git checkout <pre-phase-4-commit> -- README.md
```

### 3.2 Complete Rollback

**If migration needs to be abandoned completely:**

```bash
# Method 1: Reset to pre-migration state
git checkout main
git reset --hard pre-migration-backup
git push origin main --force  # Only if not shared!

# Method 2: Revert all migration commits
git revert <first-migration-commit>^..<last-migration-commit>

# Method 3: Create rollback branch
git checkout -b rollback-migration
git revert <all-migration-commits>
# Create PR for rollback
```

### 3.3 Partial Rollback Strategy

**Keep successful phases, rollback problematic ones:**

```bash
# Example: Keep Phases 1-2, rollback Phase 3
git checkout feature/structure-reorganization
git revert <phase-3-commits>

# Verify Phases 1-2 still work
go test ./helper/...
go test ./modules/...
go build ./cmd/...

# Continue development with partial migration
```

### 3.4 Rollback Checklist

Before rolling back, verify:
- [ ] Backup taken of current state
- [ ] Rollback branch created
- [ ] Team notified of rollback
- [ ] Reason for rollback documented
- [ ] Lessons learned captured

After rollback:
- [ ] All tests passing
- [ ] Build successful
- [ ] No orphaned files
- [ ] Import paths consistent
- [ ] Documentation reverted
- [ ] Team notified of completion

---

## 4. Risk Mitigation

### 4.1 Identified Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Import path conflicts | Medium | High | Use type aliases, gradual migration |
| Build failures | Medium | High | Test after each file move |
| Circular dependencies | Low | High | Interface-first design, careful ordering |
| Test failures | High | Medium | Run tests after each phase |
| Team disruption | Medium | Medium | Clear communication, documentation |
| Merge conflicts | High | Low | Feature branch, small commits |
| Performance regression | Low | High | Benchmark tests, profiling |

### 4.2 Mitigation Strategies

**Import Path Conflicts:**
- Create compatibility layers with type aliases
- Use package aliases during transition
- Update imports incrementally
- Automated find/replace with verification

**Build Failures:**
- Test build after every file move
- Keep compilation working at all times
- Use `go build ./...` frequently
- CI/CD pipeline catches issues early

**Circular Dependencies:**
- Define interfaces before moving
- Follow dependency rules (helper → modules → core)
- Use dependency injection
- Review import graph regularly

**Test Failures:**
- Run tests after each change
- Fix tests before proceeding
- Add migration-specific tests
- Integration tests catch issues

**Team Disruption:**
- Clear communication plan
- Migration documentation accessible
- Training sessions on new structure
- Pair programming during transition

### 4.3 Validation Checklist

**Per Phase:**
- [ ] All files moved successfully
- [ ] Package declarations updated
- [ ] Import paths updated
- [ ] Tests passing
- [ ] Build successful
- [ ] Linter clean
- [ ] Documentation updated
- [ ] Git commits clean and descriptive

**Final Validation:**
- [ ] Complete test suite passes
- [ ] All entry points build
- [ ] Integration tests pass
- [ ] No old import paths remain
- [ ] Documentation complete
- [ ] Team trained
- [ ] Performance benchmarks stable
- [ ] No regressions detected

---

## 5. Communication Plan

### 5.1 Stakeholder Communication

**Before Migration:**
- Email to team with migration plan
- Review session with tech leads
- Q&A session with developers
- Timeline communicated clearly

**During Migration:**
- Daily standup updates
- Slack channel for migration questions
- Documentation wiki updated real-time
- Blockers escalated immediately

**After Migration:**
- Migration summary report
- Lessons learned session
- Update onboarding docs
- Celebrate completion!

### 5.2 Documentation Updates

**During Migration:**
- Update `docs/architecture/` with new structure
- Create migration guide for team
- Update `README.md` with new import paths
- Update developer setup guide

**After Migration:**
- Architecture decision records (ADRs)
- Code organization guide
- Package dependency map
- Contributing guidelines update

---

## 6. Success Metrics

### 6.1 Quantitative Metrics

**Code Organization:**
- Number of packages: ~14 (from scattered structure)
- Average package size: <2000 lines
- Cyclomatic complexity: Reduced by 20%
- Import path depth: Max 3 levels

**Quality Metrics:**
- Test coverage: Maintained or improved
- Build time: No significant increase
- Linter warnings: Zero
- Go report card: A+ grade

**Dependency Metrics:**
- Circular dependencies: Zero
- Helper dependencies: Zero internal deps
- Module dependencies: Only helper deps
- Core dependencies: Modules + helpers only

### 6.2 Qualitative Metrics

**Team Productivity:**
- Time to locate code: Reduced
- Onboarding time: Reduced
- Code review time: Reduced
- Parallel development: Enabled

**Maintainability:**
- Clear component boundaries
- Obvious extension points
- Self-documenting structure
- Easy refactoring

---

## 7. Timeline Summary

**Total Duration:** 3-4 weeks (21 working days)

| Phase | Days | Risk | Rollback | Dependencies |
|-------|------|------|----------|--------------|
| Phase 0 | 1-2 | Low | Easy | None |
| Phase 1 | 3-5 | Low | Easy | Phase 0 |
| Phase 2 | 6-12 | Medium | Medium | Phase 1 |
| Phase 3 | 13-18 | High | Hard | Phase 2 |
| Phase 4 | 19-21 | Low | Easy | Phase 3 |

**Critical Path:** Phase 3 (Core Layer) is most complex  
**Buffer Time:** 2-3 days built into schedule  
**Go/No-Go Decision Points:** After Phase 1, After Phase 2

---

## 8. Conclusion

This migration strategy provides:

✅ **Phased approach** minimizing risk  
✅ **Clear rollback procedures** at each phase  
✅ **Detailed step-by-step instructions**  
✅ **Risk mitigation strategies**  
✅ **Validation checkpoints**  
✅ **Communication plan**  
✅ **Success metrics**  

**Recommended Approach:**
1. Execute Phase 0-1 (Helper layer) first - Low risk
2. Evaluate success, proceed to Phase 2 (Modules)
3. Take break, plan Phase 3 carefully (Core refactoring)
4. Execute Phase 3 with pair programming
5. Cleanup in Phase 4

**Key Success Factors:**
- Test continuously
- Communicate clearly
- Move incrementally
- Maintain working state
- Don't rush Phase 3

---

**Status:** Step 4 Complete - All Documentation Ready

**All Task 1.3 Deliverables Completed:**
1. ✅ Current structure analysis (`current_structure_analysis.md`)
2. ✅ Component categorization (`component_categorization.md`)
3. ✅ Directory structure design (`directory_structure_design.md`)
4. ✅ Migration strategy (`migration_strategy.md`)

---

**Migration Strategy Completed:** 2025-11-01  
**Analyst:** Agent_Infrastructure_Foundation  
**Ready for:** Implementation Phase (Task 1.4+)
