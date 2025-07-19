# Input Package Test Data

This directory contains test data files used by the input package tests.

## Files

### char.def
This file is copied from the Rust implementation's test resources:
- Source: `sudachi.rs/sudachi/tests/resources/char.def`
- Purpose: Test character category definitions that match the Rust test behavior
- Usage: Used by `catGrammar()` in test_helpers.go to create a grammar with character categories identical to Rust tests

This test char.def is a minimal version compared to the standard char.def, containing only basic character category definitions used in Rust tests. It ensures compatibility between Go and Rust implementations in test scenarios.

Key differences from standard char.def:
- Much smaller (35 lines vs 179 lines)
- Contains only essential character ranges
- Does not include extended Unicode ranges like CJK Extension B
- Characters not explicitly defined fall back to DEFAULT category (in Rust) or Unicode category (in Go)