# Migrate Index Storage from JSON to SQLite

## Problem

The index is stored as a flat JSON array (`swarm/index/index.json`). Every CLI invocation — even a simple `lookup` — must read and parse the entire JSON file into memory. This is fine today with only file entries, but once symbols are indexed (see `index-symbols-during-scan`), a medium project (1,000 files, ~50 symbols each) will have ~50K entries producing an 8–12 MB JSON file that gets fully deserialized on every command.

This tool is installed by users and invoked many times per session. Startup latency matters.

## Recommendation: SQLite via `modernc.org/sqlite`

**Why SQLite:**
- Query without loading the entire index into memory — only read what's needed
- Proper indexes on `name`, `kind`, `path`, `package` for O(log n) lookups instead of O(n) scans
- Incremental updates — add/remove/update individual entries without rewriting the whole file
- Single file (`swarm/index/index.db`) — no multi-file coordination
- Users can inspect the database with the standard `sqlite3` CLI if needed
- Battle-tested, universally understood

**Why `modernc.org/sqlite` specifically:**
- Pure Go — no CGo, no C compiler needed, cross-compiles cleanly on all platforms
- This is the standard choice for Go CLI tools that need embedded SQLite
- Alternative wrapper: `zombiezen.com/go/sqlite` provides a cleaner Go API on top of `modernc.org/sqlite`

**Tradeoff:** Adds ~25–30 MB to the compiled binary size. Acceptable for a developer tool.

## Schema

```sql
-- Core entries table (replaces index.json array)
CREATE TABLE entries (
    id       INTEGER PRIMARY KEY,
    name     TEXT NOT NULL,       -- symbol or file name
    kind     TEXT NOT NULL,       -- "file", "func", "type", "package", etc.
    path     TEXT NOT NULL,       -- file path relative to project root
    line     INTEGER DEFAULT 0,   -- line number (0 for file entries)
    package  TEXT DEFAULT '',     -- package/module name
    exported BOOLEAN DEFAULT 0    -- 1 if publicly exported
);

-- Indexes for the main query patterns
CREATE INDEX idx_entries_name ON entries(name);
CREATE INDEX idx_entries_kind ON entries(kind);
CREATE INDEX idx_entries_path ON entries(path);
CREATE INDEX idx_entries_package ON entries(package);
CREATE INDEX idx_entries_name_kind ON entries(name, kind);  -- for filtered lookups

-- Metadata table (replaces meta.json)
CREATE TABLE meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
-- Rows: root, scannedAt, version, fileCount, packageCount, extensions (JSON)
```

## Design

### New file: `index/store.go`

Encapsulate all database access behind a `Store` interface so the rest of the code doesn't deal with SQL directly:

```go
type Store struct {
    db   *sqlite.Conn  // or *sql.DB
    path string
}

func OpenStore(dir string) (*Store, error)           // open or create index.db
func (s *Store) Close() error

// Write operations (used by Scan)
func (s *Store) Begin() (*Tx, error)                 // start a write transaction
func (s *Store) InsertEntries(entries []Entry) error  // bulk insert within a tx
func (s *Store) SetMeta(key, value string) error
func (s *Store) Clear() error                         // truncate for full rescan

// Read operations (used by all commands)
func (s *Store) LoadAll() ([]Entry, error)            // fallback: load everything
func (s *Store) MatchName(query string, limit int) ([]Entry, error)  // LIKE-based
func (s *Store) ByKind(kind string) ([]Entry, error)
func (s *Store) ByPath(path string) ([]Entry, error)
func (s *Store) FilePaths() ([]string, error)         // SELECT DISTINCT path WHERE kind='file'
func (s *Store) Meta(key string) (string, error)
func (s *Store) EntryCount() (int, error)
```

### Changes to `index/index.go`

- `Index.Entries` becomes lazily loaded — only populated when a query actually needs the full list
- `Scan()` writes directly to the SQLite store via a transaction (bulk insert is fast)
- `Save()` becomes a no-op (data already in DB) or is removed
- `Load()` opens the store instead of parsing JSON

### Changes to query functions

Most query functions currently call `Load()` then iterate `idx.Entries`. Migrate them incrementally:

| Function | Current | After migration |
|---|---|---|
| `Match()` / `MatchScored()` | Load all, iterate with scoring | Phase 1: `LoadAll()` + same logic. Phase 2: SQL pre-filter with `LIKE` then score in Go |
| `MatchExact()` | Load all, substring match | `SELECT * FROM entries WHERE name LIKE '%q%' OR path LIKE '%q%'` |
| `FilePaths()` | Load all, deduplicate | `SELECT DISTINCT path FROM entries WHERE kind='file'` |
| `FileCount()` | Count FilePaths | `SELECT COUNT(DISTINCT path) FROM entries WHERE kind='file'` |
| `ExtensionCounts()` | Iterate FilePaths | SQL group-by on file extension |
| `Stale()` | Load file entries, compare to disk | `SELECT path FROM entries WHERE kind='file'` then compare |
| `Search()`, `Refs()`, etc. | Load entries for file list, then read files from disk | Same — these need file contents regardless |

### Migration / backward compat

- On `Load()`: if `index.db` exists, use it. If only `index.json` exists, read JSON and migrate to SQLite automatically, then delete the JSON files.
- On `Scan()`: always write to SQLite. Remove JSON writing.
- The `meta.json` data moves into the `meta` table.
- Bump `version` in meta to `"0.2.0"`.

### File changes

| File | Change |
|---|---|
| `go.mod` | Add `modernc.org/sqlite` (or `zombiezen.com/go/sqlite`) dependency |
| `index/store.go` | **New** — SQLite store implementation |
| `index/store_test.go` | **New** — Store unit tests |
| `index/index.go` | Replace JSON read/write with Store calls |
| `index/stale.go` | Use `Store.ByKind("file")` instead of loading all entries |
| `index/index_test.go` | Update tests to work with SQLite (use temp dirs) |
| `README.md` | Update storage description |
| `SKILL.md` | Update if storage details are mentioned |

### What stays the same

- The `Entry` struct — unchanged
- All CLI commands and their output — unchanged
- The `swarm/index/` directory location — `index.db` replaces `index.json` + `meta.json`
- Fuzzy scoring logic — runs in Go, not SQL
- File-reading queries (search, refs, symbols, etc.) — still read files from disk

## Phases

**Phase 1 — Store layer + Scan writes to SQLite**
- Add `store.go` with Open/Close/InsertEntries/SetMeta/LoadAll
- Change `Scan()` to write via Store instead of JSON marshal
- Change `Load()` to read via `Store.LoadAll()` — all queries work unchanged
- Auto-migrate existing JSON indexes on first load
- Tests pass with no behavior changes

**Phase 2 — Optimize hot-path queries**
- Add targeted query methods to Store (MatchName, ByKind, ByPath, FilePaths)
- Migrate `MatchExact`, `FilePaths`, `FileCount`, `ExtensionCounts`, `Stale` to use SQL
- `Match`/`MatchScored` use SQL pre-filter + Go scoring

**Phase 3 — Incremental updates (optional, future)**
- Instead of full rescan, `Scan()` checks file mod times and only re-parses changed files
- Deletes removed files, adds new files, updates modified files
- Makes re-indexing near-instant for small changes

## Dependencies

- Depends on `index-symbols-during-scan` being planned but does NOT need to be completed first — this migration works with file-only entries and will naturally support symbol entries when they're added
- No other dependencies
