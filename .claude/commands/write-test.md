# Write Tests (Learning Mode)

Help me write tests for:

**Target:** $ARGUMENTS

## Instructions for Claude:

### First, Explain Go Testing
- How `go test` works
- File naming convention (`_test.go`)
- The `testing` package basics
- How to run tests (`go test ./...`, `go test -v`, etc.)

### Table-Driven Tests
Explain the table-driven test pattern (Go's most idiomatic test style):

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty input", "", ""},
        {"normal case", "hello", "HELLO"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Something(tt.input)
            if got != tt.expected {
                t.Errorf("got %q, want %q", got, tt.expected)
            }
        })
    }
}
```

### Guide Me Through Writing
1. Help me identify what needs testing (happy path, edge cases, errors)
2. Have ME write the first test case
3. Review and improve my test
4. Suggest additional cases I might have missed

### Comparison to Ruby
- How this differs from RSpec
- No `describe`/`it` - just functions
- No magic - explicit assertions

### Don't
- Don't write all the tests for me
- Don't use third-party test frameworks (keep it stdlib)
