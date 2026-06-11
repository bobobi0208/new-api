# Probe Defense Overview Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make group probe-defense configuration visible at a glance and complete the default configurable probe signatures requested earlier.

**Architecture:** Keep the existing backend API shape and add the overview in the classic frontend by deriving table rows from groups, policies, sources, signatures, and recent events. Extend the existing seed data so configurable signatures cover the previously discussed probe topics instead of relying on hidden legacy matcher rules.

**Tech Stack:** Go, GORM, React, Semi UI, Bun/Vite classic frontend.

---

### Task 1: Default Signature Coverage

**Files:**
- Modify: `model/probe_defense_test.go`
- Modify: `model/probe_defense.go`

**Steps:**
1. Add a test that seeds default probe-defense data and asserts configurable signatures exist for tag echo, Einstein five houses, math 1+1, web search AI news, identity/medicine-bottle conflict, OCR, and long-stream/high max_tokens.
2. Run `go test ./model -run TestSeedDefaultProbeDefenseData -count=1` and confirm the new test fails before implementation.
3. Extend `SeedDefaultProbeDefenseData` with the missing default signatures.
4. Re-run the same test and confirm it passes.

### Task 2: Group Policy Overview UI

**Files:**
- Modify: `web/classic/src/pages/ProbeDefense/index.jsx`

**Steps:**
1. Add derived maps for source names, signature counts by source, and latest event by group.
2. Fetch recent probe events independently of the filtered event log.
3. Add a `Table` above the existing edit form with columns for group, enabled state, action mode, sources, rule count, target URL, API key state, latest hit, and operations.
4. Add rule pattern summaries to the feature-rule table.
5. Verify classic frontend builds with `cd web/classic && bun run build`.

### Task 3: Final Verification And Commit

**Steps:**
1. Run targeted Go tests for model/probe_defense.
2. Run the classic frontend build.
3. Commit only the plan, backend seed/test, and classic frontend changes.
4. Stop before Docker deployment and request explicit confirmation.
