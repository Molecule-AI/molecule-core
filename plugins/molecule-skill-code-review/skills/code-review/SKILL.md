---
name: code-review
description: "Review code for best practices, modularity, scalability, abstraction, test coverage, redundancy, hardcoded values, type safety, performance, naming, API design, async patterns, config/env sync, template consistency, and documentation alignment. Generates detailed report with issues and recommendations."
---

# Code Review

Perform a comprehensive code review of recent changes or specified files to ensure quality standards.

## Review Criteria

### 1. Best Practices
- Follows TypeScript strict mode conventions
- Proper error handling (try/catch, error types, no silent failures)
- No hardcoded values (use environment variables or constants)
- Proper logging with appropriate log levels
- Security best practices (input validation, no SQL injection, XSS prevention)
- No console.log in production code (use logger)

### 2. Modularity
- Single responsibility principle (each function/class does one thing)
- Functions are small and focused (< 50 lines ideally)
- No code duplication (DRY principle)
- Clear separation of concerns (routes, services, utilities)

### 3. Scalability
- Efficient database queries (proper indexing, no N+1 queries)
- Connection pooling used correctly
- Async operations handled properly
- No blocking operations in hot paths

### 4. Abstraction
- Interfaces/types defined for all public APIs
- Implementation details hidden behind abstractions
- Adapter pattern used for external services (LLM, database)
- Configuration externalized (not hardcoded)

### 5. Test Coverage
- Unit tests exist for all utility functions and service functions
- Service layer has integration tests
- Edge cases are covered
- Test files go in `tests/unit/` or `tests/integration/`, named `*.test.ts`
- All exported functions have at least one test

### 6. No Redundancy
- No duplicate code blocks (extract to shared functions/utilities)
- No repeated logic across files (consolidate into services)
- No redundant imports or unused variables
- No copy-pasted code with minor variations (use parameters/generics)
- No redundant API calls (cache or batch where appropriate)
- No repeated validation logic (create reusable validators)
- No duplicate helper logic in test files (extract shared test utilities)

### 7. No Hardcoded Values
- No hardcoded URLs, API endpoints, or hostnames (use env vars)
- No hardcoded credentials, keys, or secrets (use env vars)
- No magic numbers without named constants
- No hardcoded file paths (use configuration or path utilities)
- No hardcoded timeouts/limits (externalize to config)
- No hardcoded error messages (use constants or i18n)
- No hardcoded feature flags (use configuration system)
- No hardcoded tenant/user IDs in business logic

### 8. Type Safety
- No usage of `any` type (use `unknown` or proper types)
- Proper null/undefined handling (optional chaining, nullish coalescing)
- Generic types used appropriately
- Return types explicitly declared for public functions
- No type assertions (`as`) without validation

### 9. Performance
- No memory leaks (cleanup subscriptions, timers, event listeners)
- Proper memoization for expensive computations
- Lazy loading for heavy components/modules
- Efficient data structures for the use case
- No synchronous operations blocking the event loop
- Batch API calls where possible (e.g., single `messages.modify` with multiple label IDs)

### 10. Naming & Readability
- Descriptive variable/function names (no `x`, `temp`, `data`)
- Consistent naming conventions (camelCase, PascalCase)
- No misleading names (function does what name suggests)
- Boolean variables prefixed appropriately (`is`, `has`, `should`)
- No excessive abbreviations
- Code is self-documenting where possible

### 11. API Design
- Consistent response formats across endpoints
- Proper HTTP status codes used
- Input validation at API boundaries
- Proper error response structure
- RESTful conventions followed
- API versioning considered for breaking changes

### 12. Async & Concurrency
- No unhandled promise rejections
- Proper race condition handling
- Concurrent operations use Promise.all where appropriate
- No floating promises (missing await)
- Proper cleanup on component unmount/request abort
- AbortController used for cancellable operations

### 13. Dependency Management
- No unused dependencies in package.json
- No deprecated packages
- Security vulnerabilities addressed (npm audit)
- Peer dependency conflicts resolved
- Dependencies pinned to specific versions where needed

### 14. Environment & Configuration Sync
- Every env var used in `src/config/env.ts` is documented in `.env.example`
- Every env var in `.env.example` is defined in the Zod schema (`src/config/env.ts`)
- Default values match between `.env.example` comments and Zod `.default()` calls
- Conditional requirements are documented (e.g., "only required when LLM_PROVIDER=openai")
- No env vars referenced directly via `process.env` outside of `src/config/env.ts` and `src/lib/logger.ts`
- `docker-compose.yml` service ports/URLs align with `.env.example` defaults
- `Dockerfile` exposes the correct `PORT` matching `.env.example`
- `docs/railway-deployment.md` env var list matches the Zod schema

### 15. Template & Documentation Consistency
- Email templates in `docs/templates/` have all `{{variable}}` placeholders documented in their "Available Variables" table
- Template variable sources match actual database columns and service outputs
- Classification categories in `docs/classification-design.md` match the `EmailCategory` type in `src/types/email.ts`
- Confidence thresholds in docs match the actual thresholds implemented in code
- Sub-types in docs match the template trigger conditions
- Gmail label names in code (`GmailLabel` const) match labels documented in architecture docs
- API endpoint schemas in `docs/api-spec.md` match actual route handler request/response types
- Error handling strategies in `docs/error-handling.md` match actual retry/error class behavior (e.g., `isRetryable` flags)

### 16. Error Messages & UX
- User-friendly error messages (no technical jargon)
- Loading states for async operations
- Empty states handled gracefully
- Graceful degradation when features fail
- Confirmation for destructive actions
- Success feedback for completed actions
- Error boundaries to prevent full app crashes
- Proper form validation with clear feedback

## Output Format

```markdown
## Code Review Report

### Files Reviewed
- List of files

### Issues Found

#### 🔴 Critical
- [file:line] Description - Recommendation

#### 🟡 Warning
- [file:line] Description - Recommendation

#### 🔵 Suggestions
- [file:line] Description - Recommendation

### Config & Template Sync
- .env.example ↔ env.ts schema: [in sync / N mismatches]
- docs/classification-design.md ↔ src/types/email.ts: [in sync / N mismatches]
- docs/templates/ ↔ template variables: [in sync / N mismatches]
- docs/error-handling.md ↔ src/lib/errors.ts: [in sync / N mismatches]

### Test Coverage
- Files missing tests
- Coverage gaps

### Summary
- Total issues count
- Action items
```
