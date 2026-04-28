# Changelog

## [0.0.3-alpha] - 2026-04-28

### Added
- Backend: Initialize Go Gin backend entrypoint and module files
- Backend: Add unified API response model for HTTP handlers
- Backend: Add chat HTTP connect endpoint with cookie-based user identity check
- Backend: Add chat WebSocket endpoint and basic event protocol
- Backend: Add in-memory chat hub for room membership and message broadcast

### Changed
- Project: Update root .gitignore to ignore test directory

### Notes
- WebSocket events currently support join, message, ping, ack, pong, and error
- Current chat storage is in-memory (non-persistent), suitable for prototype phase


## [0.0.2-alpha] - 2026-04-21

### Added
- Frontend: Add home lobby demo page


## [0.0.1-alpha] - 2026-04-06

### Added

#### chore(project): Initialize frontend with Vite + Vue 3
- Set up Vite build tool with Vue 3 + TypeScript template
- Configure TypeScript and TSConfig for both app and node

#### chore(project): Add root .gitignore
- Ignore node_modules, build outputs (dist), IDE settings
- Prevent environment files and OS-specific files from being tracked

#### docs: Initialize changelog
- Set up CHANGELOG.md following semantic versioning
