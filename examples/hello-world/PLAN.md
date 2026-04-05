# Hello World PLAN

## Phase 1: Scaffold baseline
- Confirm manifest metadata, tags, and permissions for hello-world.
- Acceptance criteria: utilivo verify passes.

## Phase 2: Core workflow
- Implement the plugin's primary user flow in tool/.
- Lead with the first card (section title and description); avoid a separate page-level header and decorative chrome unless the product needs them.
- Acceptance criteria: interaction works from tool/index.html end-to-end.

## Phase 3: Quality and release
- Add or refine tests under tests/.
- Acceptance criteria: utilivo verify and utilivo test pass; ready for the maintainer to run utilivo build and inspect signed .ubb outputs after review.
