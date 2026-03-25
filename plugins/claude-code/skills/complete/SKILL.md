---
name: complete
description: Finalize a Colign change after all tasks are done. Use when implementation is finished, all tasks are marked done, and the change is ready to advance to the next workflow stage. Also use when the user says they're done with a change, want to wrap up, or asks to move a change forward.
---

# Complete a Change

Verify all work is done, check gate conditions, and advance the change to the next workflow stage.

---

## Workflow

### Step 0: Select Project & Change

1. Call `mcp__colign__list_projects` to get available projects
2. If multiple projects exist, present the list and ask the user which project to work on
3. Call `mcp__colign__list_changes` for the selected project
4. Present changes that are in progress and ask the user which change to complete
5. If the user already specified a project or change (by name or ID), skip the corresponding selection step

### Step 1: Check Task Completion

Call `mcp__colign__list_tasks` to verify all tasks are `done`.

If any tasks remain `todo` or `in_progress`:
- List the incomplete tasks
- Suggest running `/colign:implement` to finish them
- Ask if the user wants to proceed anyway

### Step 2: Check Acceptance Criteria

Call `mcp__colign__list_acceptance_criteria` to verify AC status.

If any AC are not met:
- List the unmet criteria
- Ask if these should be toggled or if work remains

### Step 3: Check Gate Conditions

Call `mcp__colign__get_gate_status` to see if the change can advance to the next stage.

Show the gate status clearly:

```
Gate → [next stage]
✓ Design document exists
✓ All tasks completed
✗ At least 1 approval needed
```

### Step 4: Summarize and Advance

Summarize what was accomplished:

```
Change: [name]
Stage: [current] → [next]
Tasks: [X/Y completed]
AC: [X/Y met]

Key deliverables:
- [deliverable 1]
- [deliverable 2]
```

If all gate conditions are met and user confirms:
- Call `mcp__colign__advance_stage` to move to the next stage

If gate conditions are not met:
- Show what's missing
- Suggest actions to resolve

### Step 5: Save Learnings (Optional)

Ask if there are project learnings worth saving:
- Call `mcp__colign__save_memory` to update project memory with conventions, decisions, or patterns discovered during implementation

### Step 6: Archive (Optional)

If the change has reached the final stage (Ready), ask whether to archive it:

```
All done! Archive this change?
→ Yes — call archive_change to close it out
→ No  — leave it open for now
```

If user confirms, call `mcp__colign__archive_change` to archive.

---

## Guidelines

- **Never advance a stage without user confirmation**
- **Always check gate conditions** — don't just check tasks
- **If tasks are incomplete**, suggest `/colign:implement` first
- **Include a summary** of what was built for team visibility
- **Save learnings** — project memory helps future changes
