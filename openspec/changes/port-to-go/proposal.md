# Proposal

Source issue: [#5 Migrate the module to a single Go binary](https://github.com/slatkin/luvus-prompt-library/issues/5).

## Why

The module needs Node (>= 22.18 or >= 23.6 for type stripping) on every machine, and Node is its last runtime requirement. Later menus (pickers, prompt preview) need a terminal UI. Go links the UI libraries into one binary, so the module can have those menus and still have zero runtime dependencies. Node needs an npm package or an external binary for the same result. This change ports the module to Go first, with the same behaviour, so that the menus change starts from the Go binary.

## What Changes

- Port every script (`lib.ts`, `refresh.ts`, `toggle.ts`, `select.ts`, `edit.ts`, `send.ts`) to one Go program, `prompt-dock`, with one verb per action (`refresh`, `toggle`, `select`, `edit-inline`, `edit-external`, `send`). The dock rows, row menus, toasts, state files, and luvus CLI calls stay the same.
- The binary uses the Go standard library only. It adds no third-party modules.
- `go install` puts the binary on `PATH`. Every manifest `command` becomes `["prompt-dock", "<verb>"]`. A `[[build]]` entry runs `go install` at git-install time. For a local `link`, the user runs `go install` by hand.
- **BREAKING**: the requirement changes from Node at runtime to a Go toolchain at install time, with the Go binary folder (`GOBIN`, by default `~/go/bin`) on the `PATH` that luvus sees.
- Linux is the tested platform. The code compiles for Windows, but Windows is not tested. No binaries are built or distributed for any platform.
- Port `lib.test.ts` to `go test`.
- Replace the npm gates (`tsc`, `eslint`, `prettier`, `node --test`) with `gofmt`, `go vet`, and `go test` in the pre-commit hook and in CI (Linux only). Remove `package.json`, `package-lock.json`, `tsconfig.json`, `eslint.config.js`, `.prettierrc.json`, and `.prettierignore`.
- Update `README.md` for the new requirements and install steps.

Out of scope:
- New menus, overlay panes, and the terminal UI libraries for them. A later change adds them.
- New row menu entries.
- Release builds, prebuilt binaries, and Windows testing.
- Any change to luvus.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- None. The `prompt-dock` and `agent-prompt-send` requirements do not name the language, the scripts, or a platform, and the behaviour they describe does not change. The change sets `skip_specs: true`.

## Impact

- Code: all `.ts` files are replaced by Go files in `cmd/prompt-dock/` (`package main`), plus `go.mod` at the repo root.
- Manifest: every `command` changes to the bare binary name, and one `[[build]]` entry is added. The module version goes to `0.2.0`.
- Users: an install needs a Go toolchain and `GOBIN` on `PATH`, and no Node.
- Tooling: the npm dev dependencies, the npm-based pre-commit hook, and the Node CI job are removed. `core.hooksPath` is no longer set by `npm install`, so the README tells developers to set it by hand.
- State: `expanded.json` and `selected.json` keep their format, so an upgrade keeps the expanded folders and the selected prompt.
