---
name: implement
description: Implement code based on a Colign spec. Use when the user says "implement this", "start coding", "work on the next task", "build this feature", or any request to write code that has a corresponding spec or task list on the Colign platform. Reads specs and tasks from Colign, writes code locally, and updates task status as work progresses.
---

# Implement from Spec

Read the spec from Colign, implement the code locally, and update task progress on the platform.

---

## Workflow

### Select Project & Change

1. Call `mcp__colign__list_projects` to get available projects
2. If multiple projects exist, present the list and ask the user which project to work on
3. Call `mcp__colign__list_changes` for the selected project
4. Present changes that have tasks (filter out empty drafts) and ask the user which change to implement
5. If the user already specified a project or change (by name or ID), skip the corresponding selection step

### Load Spec & Tasks

1. Call `mcp__colign__get_change` to see the change's current stage and metadata
2. Call `mcp__colign__read_spec` with `doc_type: design` to read the design
3. Call `mcp__colign__read_spec` with `doc_type: proposal` for additional context
4. Call `mcp__colign__list_tasks` to see the task list and their current status
5. Call `mcp__colign__list_acceptance_criteria` to see what needs to be verified

If no design or tasks exist, suggest running `/colign:plan` first.

### Task Loop

For each task:

1. Pick the next `todo` task (or let the user choose)
2. Call `mcp__colign__update_task` to set it to `in_progress`
3. Implement the code locally following the spec
4. Verify the implementation (build, tests, lint)
5. Call `mcp__colign__update_task` to set the task to `done`
6. Continue to next task or pause for user input

### Mid-Implementation Checks

Between tasks, periodically:
- Call `mcp__colign__get_gate_status` to see progress toward next stage
- Call `mcp__colign__list_acceptance_criteria` to check which AC are met
- Call `mcp__colign__toggle_acceptance_criteria` when an AC is satisfied by the implementation

---

## Guidelines

- **Always read the spec before coding** — don't guess requirements
- **Update task status in real-time** — the platform should reflect actual progress
- **Follow existing code conventions** — check the local codebase for patterns
- **Write tests alongside implementation** — not after
- **If the spec is unclear**, suggest running `/colign:propose` or `/colign:plan` to update it
- **If a task is too large**, suggest breaking it down via `/colign:plan`

## Verification

Before marking a task as `done`:
- Code compiles without errors
- Tests pass
- Implementation matches the spec requirements

## Next Step

When all tasks are complete, suggest running `/colign:complete` to finalize the change.
