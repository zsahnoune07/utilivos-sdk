# AGENTS.md

Use this file as the operating contract for this plugin project.
This template is based on the Utilivo AGENTS contract.

## Plugin context
- Name: Hello World
- Slug: hello-world

## Mission
Build and maintain a production-minded plugin with strict manifest-driven behavior.
Infer practical defaults when requirements are missing, document assumptions, and keep moving.

## Core behavior
1. Implement and validate; defer packaging to the maintainer unless they explicitly ask you to build.
2. Prefer complete vertical slices over partial work.
3. Keep code simple, modular, and maintainable.
4. Avoid unnecessary dependencies or abstractions.
5. Treat security, data integrity, and developer experience as first-class concerns.
6. Never leave placeholder architecture unimplemented when implementation is practical.
7. Keep plugin runtime and packaging strictly manifest-driven.

## Required workflow
- Run utilivo verify and utilivo test when making substantive changes (recommended before hand-off).
- Maintainers run utilivo build to produce release artifacts after review.
- Autonomous agents must not run utilivo build, utilivo pack, or signing steps unless the user explicitly requests it.
- Keep manifest.json as the source of truth.
- Publish signed .ubb bundles for distribution (maintainer workflow).

## Manifest contract rules
- Keep required fields valid and complete.
- tier must be free or premium (Pro in the host app; keep the literal premium value in manifests).
- category must be from controlled allowed values.
- tags must be non-empty and searchable.
- locales must include en and default_locale must be en.
- permissions must be least-privilege and explicitly declared.
- entry, icon, thumb, and screenshots must reference real files.
- min_app_version and api_version must be intentional and valid.

## PLUGIN UI RULES
- Plugins must use utilivo-ui components.
- Plugins cannot hardcode product colors or redefine --uv-* tokens.
- Plugins must support light and dark mode through host-injected theme tokens.
- Plugins must rely on CSS variables supplied by the host shell.
- Plugins must keep user-facing strings under tool/locales/ with English as default.
- Lead with the primary workflow inside the first card using uv-section__title and uv-section__description. Omit a separate uv-tool-page__header unless the product needs it. Do not add eyebrow labels or decorative pill rows unless the tool has an explicit product requirement.
- Use UtilivoRuntime.syncState for host session persistence; do not replace window.utilivo without forwarding persistState and loadState.

## Security baseline
- Never commit private keys, tokens, or env secrets.
- Validate external input and sanitize rendered content.
- Avoid unsafe dynamic execution.
- Keep permissions minimal and explicit.

## Definition of done
- Plugin can be verified and tested end-to-end; release build is the maintainer step after review.
- Manifest is valid and enforced by runtime/build workflows.
- README and tests are updated with behavior changes.
- Release artifacts are deterministic .ubb packages and signed when required (maintainer).
