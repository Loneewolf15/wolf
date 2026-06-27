# Wolf Package Registry Specification

This document defines the specification for the Wolf Package Registry, the `wolf.json` manifest format, and the `wolf install` dependency resolution process.

## 1. Registry Architecture

Wolf uses a **Decentralized Git-First** registry model with a thin vanity-URL index.

- **No Tarballs**: The registry does not host or proxy package tarballs. All packages are downloaded directly from their canonical Git repositories.
- **Vanity Index**: The registry (`registry.wolf-lang.org`) simply maps human-readable package names (e.g., `company/package`) to Git repository URLs.
- **Direct Fallback**: If a dependency URL contains a known Git host (e.g., `github.com/user/repo`), `wolf install` skips the registry and clones it directly.

## 2. Registry API

The vanity index exposes a simple JSON API.

**Endpoint**: `GET https://registry.wolf-lang.org/pkg/{namespace}/{name}`

**Response format (200 OK)**:
```json
{
  "name": "namespace/name",
  "repository": "https://github.com/user/repo",
  "latest": "v1.2.0",
  "versions": ["v1.0.0", "v1.1.0", "v1.2.0"]
}
```

If the package is not found, the registry returns `404 Not Found`.

## 3. Package Manifest (`wolf.json`)

Every Wolf package must have a `wolf.json` at its root.

```json
{
  "name": "mycompany/mypackage",
  "version": "1.2.0",
  "description": "A brief description of what this does",
  "dependencies": {
    "wolf-lang.org/x/http": "^1.0.0",
    "github.com/user/repo": "v2.1.0",
    "gitlab.com/other/lib": "~1.5.3"
  }
}
```

### Dependency Resolution Rules
The `dependencies` map uses SemVer constraints.

| Constraint | Meaning | Example Match |
|---|---|---|
| `exact` | Must match the exact version | `1.2.3` |
| `^` (Caret) | Compatible with version (no major changes) | `^1.2.3` matches `>=1.2.3 <2.0.0` |
| `~` (Tilde) | Patch updates only | `~1.2.3` matches `>=1.2.3 <1.3.0` |
| `>` or `<` | Standard bounds | `>=1.2.0`, `<2.0.0` |
| `latest` | The highest SemVer tag available | Matches highest tag |
| `branch` | A specific git branch (fallback) | `main` |

## 4. `wolf.lock` File

The lock file guarantees reproducible builds by pinning exact Git SHAs.

```json
{
  "locked": {
    "wolf-lang.org/x/http": {
      "version": "v1.0.5",
      "sha": "a1b2c3d4e5f6g7h8i9j0"
    },
    "github.com/user/repo": {
      "version": "v2.1.0",
      "sha": "f1e2d3c4b5a6"
    }
  }
}
```

## 5. `wolf install` Workflow

When a developer runs `wolf install`:

1. **Parse `wolf.json`**: Reads the explicit dependency constraints.
2. **Resolve Coordinates**:
   - For each dependency, if it's not a direct URL, query `registry.wolf-lang.org`.
   - Retrieve the Git repository URL and the list of available SemVer tags.
3. **Evaluate SemVer**: Find the highest Git tag that satisfies the constraint.
4. **Check `wolf.lock`**: If the package is already locked to a SHA that satisfies the constraint, use the locked SHA.
5. **Clone & Checkout**: Clone the Git repository into `.wolf_modules/{name}` and checkout the precise tag or SHA.
6. **Recurse**: Read the `wolf.json` of the newly fetched package and repeat the process for sub-dependencies.
7. **Write Lockfile**: Update and save `wolf.lock`.

## 6. Package Structure Constraints

To be a valid Wolf package:
- Must contain a `wolf.json`.
- Cannot use the `wolf_*` reserved core namespace.
- Source files (`.wolf`) should be placed in `src/` or the root directory.

