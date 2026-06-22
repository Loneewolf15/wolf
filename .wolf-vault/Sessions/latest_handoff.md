# Handoff — 2026-06-22 (Session 33)

## Where We Left Off
We successfully updated `wolf build` and `wolf dev` to support zero-argument invocations (defaulting to `src/main.wolf` or `main.wolf`). We also resolved a localized 10-minute timeout hang in `upload_test.go` by properly adding the necessary server startup print statement in `_server_upload.wolf`. Crucially, we drafted the **implementation plan** for Native Project AutoDiscovery in `wolf-self`, which is fully outlined in the artifacts directory.

## Commits This Session
*(Pending execution: standard commit will wrap up the CLI updates and bug log.)*

## Tests Status
- `/internal/...`: Pass
- `/e2e/...`: Fail (1 known E2E failure: `TestFileUpload`)
  - `TestFileUpload` fails with a 400 Bad Request because `wolf_http_req_file` C runtime expects a `const char*` but receives a `ptr` (Wolf Map structure) from the LLVM emitter. It is now failing fast (~20s) instead of hanging.

## Next Immediate Task
1. Review and approve the `implementation_plan.md` artifact to build out `--project <dir>` support natively inside `src/compiler/main.wolf`.
2. Once approved, execute the plan to upgrade the self-hosted compiler from single-file isolation to full directory resolution.

## Open Issues / Watch Out For
- **`wolf_http_req_file` Signature Mismatch:** The native compiler's LLVM emitter correctly types the `wolf_http_req_file` arguments as `(i64, ptr)`, but the C runtime expects a raw `const char*`. A string coercion extraction helper (like `wolf_req_strdup` extraction from maps) or an LLVM fix will be required before file uploads work natively.
- **Go Test Timeouts:** When a C runtime server component hangs or fails to emit expected stdout markers, tests will block until the 10m Go limit. Always ensure C runtime / Wolf scripts explicitly output expected synchronization markers.

## Relevant Files Modified This Session
- `cmd/wolf/main.go`
- `cmd/wolf/dev.go`
- `e2e/upload_test.go`
- `e2e/testdata/_server_upload.wolf`
- `artifacts/implementation_plan.md`
