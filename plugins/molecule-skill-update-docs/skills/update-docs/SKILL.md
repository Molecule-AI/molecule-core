---
name: update-docs
description: "Review recent edits and update all documentation including architecture docs, API specs, and edit history. Creates missing docs for new implementations."
---

# Update Documentation

Review recent code changes and update ALL relevant documentation in the `/docs` folder.

## Steps

1. **Read today's edit history**

   - Check `docs/edit-history/` for the current date's session file
   - Identify all files that were modified

2. **Analyze changes**

   - Read the modified files to understand what changed
   - Categorize changes: new features, bug fixes, architecture changes, API changes, config changes

3. **Update edit-history session file**

   - Add a summary section at the top describing what was accomplished
   - Group related changes under descriptive headings
   - Add any missing context about why changes were made

4. **Update CLAUDE.md if needed**

   - New commands or scripts added
   - Architecture or key modules changed
   - New environment variables required
   - New routes or endpoints added
   - Test counts when new test files were added

5. **Update PLAN.md (repo root) if needed**

   - When a planned phase ships, mark it complete and add any follow-ups
   - When new architectural decisions are made, update the relevant phase
   - Keep the current status / next steps section in sync with reality
   - If a feature was reverted, document the reversal and reasoning

6. **Update README.md (repo root) if needed**

   - New features visible to users (canvas tabs, deploy flows, etc.)
   - Changed setup or quickstart instructions
   - Updated tech stack list (when adding/removing major dependencies)
   - Updated test counts in the status badges
   - License or branding changes

7. **Update README.zh-CN.md (repo root) if README.md was updated**

   - Mirror any user-visible changes from README.md
   - Keep the Chinese translation in sync — don't let it drift
   - Update the same sections in both files (status, features, setup, license)

8. **Update .env.example (repo root) if needed**

   - Every new env var read by code must be documented in `.env.example`
   - Include a comment describing the var and its expected format
   - When removing an env var from code, remove from `.env.example`
   - Keep default values consistent with code defaults

9. **Update docs/README.md if needed**

   - New features or capabilities
   - Changed setup instructions
   - Updated project overview

10. **Update docs/ files**
    Review and update all architecture documentation to match current implementation

    **For each doc:**

    - Check if documented features match actual code implementation
    - Update outdated sections to reflect current code
    - Add NEW sections for features that are implemented but not documented
    - Remove or mark deprecated features that no longer exist
    - Ensure code examples match actual implementation

11. **Create new docs if needed**

    - If a significant new feature or module was added but has no documentation, create appropriate documentation
    - Follow existing documentation style and structure

12. **Report summary**
    - List all documentation files updated
    - Note any new documentation files created
    - Summarize key changes documented
