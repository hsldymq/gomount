# Unified Mount TUI Design

## Goal

Allow the `mount` subcommand's TUI to handle both mounting and unmounting, while preserving `mount <name>` and `umount <name>` CLI modes for scripting.

## Background

Currently `mount` TUI only mounts unmounted entries, and `umount` TUI only unmounts mounted entries. The user wants a single TUI where they can manage both actions.

## SelectionState Enum

```go
type SelectionState int
const (
    SelNone    SelectionState = iota // no action
    SelMount                         // will be mounted (unmounted entry toggled by user)
    SelUnmount                       // will be unmounted (mounted entry toggled by user)
)
```

## SelectorModel Changes

### Field Changes

- `SelectedMap` type: `map[int]bool` → `map[int]SelectionState`
- New field: `InitialMountState map[int]bool` — records `entry.IsMounted` at TUI init

### Space Key Logic

Unmounted entry (not in InitialMountState):
```
SelNone  → space → SelMount
SelMount → space → SelNone
```

Mounted entry (in InitialMountState):
```
not in map (keep mounted) → space → SelUnmount
SelUnmount                → space → SelNone
SelNone                   → space → SelUnmount
```

### Enter Key Logic

- Collect entries with `SelMount` → `ToMount` list
- Collect entries with `SelUnmount` → `ToUnmount` list
- If both empty → exit without action, `Cancelled = false`, `ToMount` and `ToUnmount` both nil

### Rendering

| Condition | Display | Color | Meaning |
|-----------|---------|-------|---------|
| Mounted + not in SelectedMap (keep) | ✓ | green (42) | already mounted, no action |
| `SelMount` | ✓ | cyan (86) | will mount |
| `SelUnmount` | ✓ | cyan (86) | will unmount |
| `SelNone` or unmounted + not in map | `[ ]` | default | no action |

### New Styles

- `CheckMarkMountedStyle` — green (42) checkmark for "keep mounted"
- `CheckMarkPendingStyle` — cyan (86) checkmark for "pending action" (mount or unmount)

## Return Value

```go
type MountActionResult struct {
    ToMount   []*config.MountEntry
    ToUnmount []*config.MountEntry
    Cancelled bool
}
```

New public function `SelectMountAction(mounts)` replaces `SelectMountEntry(mounts)`.

## runMount Command Changes (`cmd/gomount/main.go`)

No-arg flow:
1. Call `SelectMountAction(cfg.Mounts)` → get `MountActionResult`
2. If `Cancelled` → exit
3. If both `ToMount` and `ToUnmount` empty → exit (no-op)
4. `EnsureSudoCached()`
5. Execute unmounts first (`ToUnmount`)
6. Execute mounts (`ToMount`)
7. Print summary

With-arg flow: unchanged.

## What Stays the Same

- `umount` subcommand — fully preserved including its own TUI
- `SelectUnmountEntry()` — unchanged
- `SelectEntry()` — unchanged (generic selector)
- `ListModel` / `DisplayList()` — unchanged
- All driver logic — unchanged
- `SelectionResult` struct — kept for `SelectUnmountEntry` and `SelectEntry`

## Files Modified

1. `internal/tui/selector.go` — SelectionState enum, SelectorModel changes, rendering, `SelectMountAction()`
2. `internal/tui/styles.go` — new checkmark styles
3. `cmd/gomount/main.go` — `runMount` logic for unified mount/unmount
