---
name: plan
description: Generate a detailed implementation plan and tasks from a Colign proposal. Use when the user says "plan this out", "break this down into tasks", "how should we implement this", "create an implementation plan", or after a proposal exists and needs to be turned into actionable architecture decisions and ordered tasks on the platform.
---

# Plan Implementation

Read the proposal from Colign, design the implementation approach, and create ordered tasks. This is the bridge between "what to build" (proposal) and "building it" (implement).

---

## Workflow

### Phase 0: Select Project & Change

1. Call `mcp__colign__list_projects` to get available projects
2. If multiple projects exist, present the list and ask the user which project to work on
3. Call `mcp__colign__list_changes` for the selected project
4. Present changes and ask the user which change to plan
5. If the user already specified a project or change (by name or ID), skip the corresponding selection step

### Phase 1: Read Context

1. Call `mcp__colign__get_change` to see the change's current stage
2. Call `mcp__colign__read_spec` with `doc_type: proposal` to read the proposal
3. Call `mcp__colign__get_memory` for project conventions
4. Analyze the local codebase to understand existing patterns and constraints

If no proposal exists, suggest running `/colign:propose` first.

### Phase 2: Write the Design

Draft an implementation plan grounded in the actual codebase:

```markdown
## Architecture
High-level approach and key decisions.

## Data Model
New or modified data structures.

## Implementation Steps
1. Step one — description
2. Step two — description

## Testing Strategy
How to verify the implementation.
```

Present the design to the user for review. After approval, save it:

```
mcp__colign__write_spec  →  doc_type: "design"
```

### Phase 3: Create Tasks

Break the design into ordered implementation tasks. For each task, call:

```
mcp__colign__create_task  →  change_id, title, description, status: "todo"
```

Each task should be:
- Small enough to complete in a single session
- Independently verifiable
- Ordered by dependency

### Phase 4: Summary

```
Design: ✓ saved
Tasks: [N] created

Ready for implementation. Run /colign:implement to start.
```

---

## Guidelines

- **Always read the proposal first** — don't plan in a vacuum
- **Reference the codebase** — ground the design in existing code patterns
- **Keep tasks small** — each should be completable in one session
- **Pause between phases** — let the user review the design before creating tasks
- **Skip what exists** — if a design already exists, go straight to task creation

---

## Resuming

If design already exists but no tasks:
- Read the existing design via `read_spec`
- Skip to Phase 3 (task creation)

---

## Next Step

After tasks are created, suggest running `/colign:implement` to start coding.
