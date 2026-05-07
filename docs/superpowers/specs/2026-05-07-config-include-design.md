# Config Include & Sorting Design

## Background

Users want to share mount configurations across machines via synced directories (e.g. Syncthing, Dropbox). The main config file at `~/.config/gomount.yaml` should be able to reference other config files, so shared entries live in the sync directory and only a one-line `include` is needed per machine.

## Config Structure Changes

### `Config` struct

```go
type Config struct {
    Mounts  []MountEntry  `yaml:"mounts" mapstructure:"mounts"`
    Include []string      `yaml:"include,omitempty" mapstructure:"include"`
    Sorting *SortingConfig `yaml:"sorting,omitempty" mapstructure:"sorting"`
}
```

Changes from current:
- `Mounts` validation tag changes from `validate:"required,min=1"` to no required constraint (empty config is valid).
- `Include` is a new optional string array. Each element is a file path or glob pattern.
- `Sorting` is a new optional block that controls mount entry ordering.

### `SortingConfig` struct

```go
type SortingConfig struct {
    By StringOrSlice `yaml:"by" mapstructure:"by"`
}
```

`By` accepts either a single string or a string array in YAML (custom `StringOrSlice` type handles both forms). Each element is a field name, optionally prefixed with `-` for descending order.

Currently supported sort fields:
- `name` — sort alphabetically by entry name. Useful with namespace-style naming (e.g. `work.nas.docs`, `home.router.photos`).

More fields can be added later.

### `StringOrSlice` type

Custom type that implements `yaml.Unmarshaler` and `mapstructure` decode hooks to accept both `by: name` (string) and `by: [name, -type]` (array).

## YAML Examples

### Main config with include and sorting

```yaml
mounts:
  - name: local-dev
    type: sshfs
    mount_dir_path: ~/mnt/dev
    sshfs:
      host: devserver
      remote_path: /home/user/code

include:
  - ~/Sync/gomount.d/*.yaml
  - ./extra-mounts.yaml

sorting:
  by: name
```

### Multi-field sorting

```yaml
sorting:
  by:
    - name
    - -type
```

### Pure include config (no local mounts)

```yaml
include:
  - ~/Sync/gomount.d/*.yaml
```

### Included file format (same as normal config)

```yaml
mounts:
  - name: work.nas.docs
    type: smb
    mount_dir_path: ~/mnt/work-docs
    smb:
      addr: nas.work.example.com
      share_name: docs
      username: user
```

## Loading Flow

`LoadConfig` is refactored into a recursive loader:

```
LoadConfig(path)
  └─ loadRecursive(path, visited={})
       │
       1. Parse YAML file into Config struct
       │
       2. Process Include list:
       │   a. Expand ~ → $HOME
       │   b. Resolve relative paths → absolute (relative to current file's directory)
       │   c. filepath.Glob to expand wildcards
       │   d. For each matched file:
       │      - Compute absolute path
       │      - If already in visited → skip (cycle prevention)
       │      - Recursively loadRecursive(file, visited)
       │      - Append resulting Mounts to current Config
       │
       3. Name conflict detection:
       │    Build a map of all entry names. If a duplicate is found,
       │    return error indicating which files contributed the conflict.
       │
       4. Apply sorting (if Sorting is configured and By is non-empty)
       │
       5. Validate driver configs for all Mounts
       │
       6. Normalize all Mounts (path expansion etc.)
       │
       7. Return merged *Config
```

### Path Resolution Rules

| Input | Resolution |
|-------|-----------|
| `~/Sync/gomount.yaml` | Expand `~` to `$HOME` |
| `./team-mounts.yaml` | Relative to the including file's directory |
| `/absolute/path.yaml` | Used as-is |
| `~/Sync/gomount.d/*.yaml` | Expand `~` then apply `filepath.Glob` |

Glob patterns are resolved relative to the including file's directory after `~` expansion.

### Visited Set

A `map[string]bool` keyed by absolute file path. Before loading a file, resolve it to an absolute path and check the set. If present, skip it silently. This prevents infinite loops from circular includes.

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Include path does not exist (single file) | Warning to stderr, continue loading |
| Glob pattern matches nothing | Silent skip (normal case) |
| Circular include detected | Skip already-visited file silently |
| Name conflict across files | Error, abort — message includes conflicting name and source files |
| Included file has parse/validation errors | Error, abort |
| Include path has permission denied | Warning to stderr, continue loading |
| Main config has no mounts and no include | Valid — empty config |
| Main config has no mounts, include files don't exist | Valid — empty config, warnings printed |

Warnings are printed to stderr with format: `WARNING: included file not found: <path>`

## Sorting

- Implicitly enabled: `sorting` block present + `by` non-empty = sorting active.
- `sorting` is only effective in the main (top-level) config file. If included files contain a `sorting` block, it is silently ignored.
- Omit `sorting` entirely to preserve declaration order.
- Sort is applied after all mounts are merged, before validation/normalization.
- Per-field direction: `-` prefix means descending. No prefix means ascending.
- Sort fields beyond `name` can be added in future without breaking existing configs.

## Impact on Existing Commands

No changes needed for `list`, `mount`, `unmount`, `interactive` — they consume `Config.Mounts` and are agnostic to entry source.

`config-example` command should update the embedded example YAML to include an `include` and `sorting` usage section (commented out).

## Implementation Scope

Files to modify:
- `internal/config/types.go` — add `Include`, `Sorting`, `StringOrSlice` types
- `internal/config/config.go` — refactor `doLoad` into recursive loader, add sorting logic, relax validation
- `internal/config/types_test.go` — new tests for include, merge, conflict, sorting, cycle detection
- `cmd/gomount/config_example.yaml` — update example

Files that remain unchanged:
- All driver code (`internal/drivers/`)
- All TUI code (`internal/tui/`)
- All interaction code (`internal/interaction/`)
- `cmd/gomount/main.go` (no changes to command wiring)
