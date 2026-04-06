# AGENTS.md

Use this file as the operating contract for this plugin project.
This template is based on the Utilivos AGENTS contract.

## Plugin context
- Name: Invalid Missing Tags
- Slug: invalid-missing-tags

## Mission
Build and maintain a production-minded plugin with strict manifest-driven behavior.
Infer practical defaults when requirements are missing, document assumptions, and keep moving.

## Core behavior
1. Build first, ask rarely.
2. Prefer complete vertical slices over partial work.
3. Keep code simple, modular, and maintainable.
4. Avoid unnecessary dependencies or abstractions.
5. Treat security, data integrity, and developer experience as first-class concerns.
6. Never leave placeholder architecture unimplemented when implementation is practical.
7. Keep plugin runtime and packaging strictly manifest-driven.

## Required workflow
- Run utilivos verify before every build.
- Run utilivos test before every build.
- Use utilivos build to produce release artifacts.
- Keep manifest.json as the source of truth.
- Publish signed .ubb bundles for distribution.

## Manifest contract rules
- Keep required fields valid and complete.
- tier must be `free` or `premium` (Pro in the host app; keep literal `premium` in manifests).
- category must be from controlled allowed values.
- tags must be non-empty and searchable.
- locales must include en and default_locale must be en.
- permissions must be least-privilege and explicitly declared.
- entry, icon, thumb, and screenshots must reference real files.
- min_app_version and api_version must be intentional and valid.

## PLUGIN UI RULES
- Plugins must use utilivos-ui components.
- Plugins cannot hardcode colors.
- Plugins must support light/dark mode.
- Plugins must rely on CSS variables.

## Security baseline
- Never commit private keys, tokens, or env secrets.
- Validate external input and sanitize rendered content.
- Avoid unsafe dynamic execution.
- Keep permissions minimal and explicit.

## Definition of done
- Plugin can be verified, tested, built, and inspected end-to-end.
- Manifest is valid and enforced by runtime/build workflows.
- README and tests are updated with behavior changes.
- Artifacts are deterministic .ubb packages and signed when required.
