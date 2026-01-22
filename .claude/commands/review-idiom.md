# Review for Idiomatic Go

Review this code for idiomatic Go patterns:

**Target:** $ARGUMENTS

## Instructions for Claude:

Analyze the code and provide feedback on:

### 1. Error Handling
- Is error handling explicit and complete?
- Are errors wrapped with context where appropriate?
- Any swallowed errors?

### 2. Naming Conventions
- Are variable names Go-idiomatic? (short in small scope, descriptive in large)
- Do exported names make sense from the caller's perspective?
- MixedCaps, not snake_case?

### 3. Code Organization
- Is the code doing one thing well?
- Should anything be split into smaller functions?
- Are there any "god functions"?

### 4. Go-specific Patterns
- Could this use interfaces better?
- Is there unnecessary allocation (e.g., building strings in loops)?
- Are slices being used efficiently?

### 5. Ruby Developer Pitfalls
- Anything that looks Ruby-ish but isn't Go-idiomatic?

For each suggestion, explain WHY it's more idiomatic - don't just say "do this instead".
