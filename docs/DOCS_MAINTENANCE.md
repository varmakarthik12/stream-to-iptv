# Documentation Maintenance Checklist & Guidelines

Whenever modifying the `stream-to-iptv` codebase, follow this checklist to ensure that all end-user documentation, API references, Docker guides, and screenshots remain accurate and synchronized.

---

## Developer Change Checklist

Before submitting a Pull Request or pushing a release tag, verify each item:

### 1. Database & Schema Changes
- [ ] Has a new migration file been added to `internal/db/migrations/` (e.g. `00X_description.sql`)?
- [ ] Are any new fields reflected in `internal/models/models.go` and `web/src/api.ts`?
- [ ] If new persistent assets or directories are introduced, are they strictly placed under `DATA_DIR` (`./data`)?
- [ ] Has the backup/restore section in `docs/features-and-guide.md` been updated if table dependencies changed?

### 2. Stream Engine & FFmpeg Settings
- [ ] Were new FFmpeg command-line flags or input parameters added?
- [ ] If new stream modes, watchdog rules, or recovery algorithms were introduced, are they explained in:
  - `README.md` (Feature highlights)
  - `docs/features-and-guide.md` (Deep dive section)
- [ ] Has `StreamEditorModal.tsx` been tested for seamless inline creation?

### 3. Frontend & UI Changes
- [ ] Were new pages or route endpoints added?
- [ ] If navigation items changed, are they reflected in `web/src/components/Navbar.tsx` and user documentation?
- [ ] Did you test building the UI bundle via `npm --prefix web run build`?

### 4. Docker & Volume Mounts
- [ ] Ensure that no new local storage paths are created outside the designated `DATA_DIR` root directory.
- [ ] Verify that the Docker run command (`-v /path/to/data:/app/data`) persists all assets without data loss upon container recreation.
- [ ] If new ports or network protocols are needed, update `EXPOSE` in `docker/Dockerfile` and documentation.

### 5. Release Pipeline
- [ ] Did dependencies change? Verify `go.mod`, `web/package.json`, and `.goreleaser.yaml`.
- [ ] Ensure `CGO_ENABLED=0` remains functional so multi-arch binaries compile cleanly across Linux, macOS, and Windows.

---

## Documentation File Map

| File | Purpose | When to Update |
| :--- | :--- | :--- |
| `README.md` | Primary repository landing page, quickstart, Docker commands, and binary downloads. | Update when installation steps, major features, or volume mount commands change. |
| `docs/features-and-guide.md` | Comprehensive manual explaining all features (streams, categories, logos, EPG, troubleshooting, security). | Update whenever UI workflows, stream lifecycle options, or EPG translation features change. |
| `docs/DOCS_MAINTENANCE.md` | This reminder checklist. | Update if project maintenance policies or release steps change. |
