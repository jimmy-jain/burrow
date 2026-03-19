package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- Tool schema tests ---

func TestToolDefinitions(t *testing.T) {
	tools := []struct {
		name        string
		tool        mcp.Tool
		wantParams  []string
		destructive bool
	}{
		{
			name:       "burrow_status",
			tool:       toolStatus(),
			wantParams: nil,
		},
		{
			name:       "burrow_analyze",
			tool:       toolAnalyze(),
			wantParams: []string{"path"},
		},
		{
			name:       "burrow_dupes",
			tool:       toolDupes(),
			wantParams: []string{"path"},
		},
		{
			name:       "burrow_doctor",
			tool:       toolDoctor(),
			wantParams: nil,
		},
		{
			name:       "burrow_size",
			tool:       toolSize(),
			wantParams: nil,
		},
		{
			name:       "burrow_report",
			tool:       toolReport(),
			wantParams: nil,
		},
		{
			name:       "burrow_clean_preview",
			tool:       toolCleanPreview(),
			wantParams: nil,
		},
		{
			name:        "burrow_clean_execute",
			tool:        toolCleanExecute(),
			wantParams:  []string{"confirmed"},
			destructive: true,
		},
		{
			name:        "burrow_dupes_conserve",
			tool:        toolDupesConserve(),
			wantParams:  []string{"confirmed", "conserve_dir", "path"},
			destructive: true,
		},
		{
			name:       "burrow_dupes_restore",
			tool:       toolDupesRestore(),
			wantParams: []string{"conserve_dir"},
		},
	}

	for _, tt := range tools {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tool.Name; got != tt.name {
				t.Errorf("tool name = %q, want %q", got, tt.name)
			}

			if tt.tool.Description == "" {
				t.Error("tool description should not be empty")
			}

			// Marshal to JSON to inspect the schema structure.
			data, err := json.Marshal(tt.tool)
			if err != nil {
				t.Fatalf("marshal tool: %v", err)
			}

			var schema map[string]any
			if err := json.Unmarshal(data, &schema); err != nil {
				t.Fatalf("unmarshal tool schema: %v", err)
			}

			// Verify required parameters are present in the schema.
			if tt.wantParams != nil {
				inputSchema, ok := schema["inputSchema"].(map[string]any)
				if !ok {
					t.Fatal("inputSchema missing or wrong type")
				}
				properties, ok := inputSchema["properties"].(map[string]any)
				if !ok {
					t.Fatal("properties missing or wrong type")
				}
				for _, param := range tt.wantParams {
					if _, exists := properties[param]; !exists {
						t.Errorf("expected parameter %q not found in schema", param)
					}
				}
			}

			// Verify annotations for destructive tools.
			if tt.destructive {
				annotations, ok := schema["annotations"].(map[string]any)
				if !ok {
					t.Fatal("annotations missing for destructive tool")
				}
				if dh, ok := annotations["destructiveHint"].(bool); !ok || !dh {
					t.Error("destructive tool should have destructiveHint=true")
				}
			}
		})
	}
}

func TestToolCount(t *testing.T) {
	// Verify the expected number of tools are registered.
	const expectedTools = 10

	srv := newTestServer()
	tools := srv.ListTools()
	if len(tools) != expectedTools {
		t.Errorf("registered tools = %d, want %d", len(tools), expectedTools)
	}
}

// --- Executor tests ---

func TestResolveBinaryFromEnv(t *testing.T) {
	// Create a temporary fake binary.
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "burrow")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envBurrowPath, fakeBin)

	exec, err := NewExecutor()
	if err != nil {
		t.Fatalf("NewExecutor() unexpected error: %v", err)
	}
	if exec.BinaryPath() != fakeBin {
		t.Errorf("BinaryPath() = %q, want %q", exec.BinaryPath(), fakeBin)
	}
}

func TestResolveBinaryMissing(t *testing.T) {
	// Point to a nonexistent path and ensure PATH lookup also fails.
	t.Setenv(envBurrowPath, "/nonexistent/burrow")
	t.Setenv("PATH", t.TempDir())

	_, err := NewExecutor()
	if err == nil {
		t.Fatal("NewExecutor() expected error for missing binary")
	}
}

func TestExecutorRunSuccess(t *testing.T) {
	// Create a fake burrow binary that outputs JSON.
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "burrow")
	script := `#!/bin/sh
echo '{"status":"ok"}'
`
	if err := os.WriteFile(fakeBin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envBurrowPath, fakeBin)

	exec, err := NewExecutor()
	if err != nil {
		t.Fatalf("NewExecutor() error: %v", err)
	}

	result, err := exec.Run(context.Background(), "status", "--json")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "{\"status\":\"ok\"}\n" {
		t.Errorf("Stdout = %q, want JSON output", result.Stdout)
	}
}

func TestExecutorRunFailure(t *testing.T) {
	// Create a fake burrow binary that exits with an error.
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "burrow")
	script := `#!/bin/sh
echo "something went wrong" >&2
exit 1
`
	if err := os.WriteFile(fakeBin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envBurrowPath, fakeBin)

	exec, err := NewExecutor()
	if err != nil {
		t.Fatalf("NewExecutor() error: %v", err)
	}

	result, err := exec.Run(context.Background(), "clean")
	if err != nil {
		t.Fatalf("Run() should not return error for nonzero exit: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if result.Stderr == "" {
		t.Error("Stderr should contain error message")
	}
}

// --- Handler tests ---

func TestDestructiveToolRequiresConfirmation(t *testing.T) {
	fakeBin := createFakeBurrow(t, `#!/bin/sh
echo '{"cleaned":true}'
`)

	t.Setenv(envBurrowPath, fakeBin)
	exec, err := NewExecutor()
	if err != nil {
		t.Fatal(err)
	}

	handler := handleCleanExecute(exec)

	// Call without confirmed parameter.
	req := newCallToolRequest("burrow_clean_execute", map[string]any{})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result when confirmed is not set")
	}

	// Call with confirmed=false.
	req = newCallToolRequest("burrow_clean_execute", map[string]any{"confirmed": false})
	result, err = handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result when confirmed=false")
	}

	// Call with confirmed=true.
	req = newCallToolRequest("burrow_clean_execute", map[string]any{"confirmed": true})
	result, err = handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Error("expected success when confirmed=true")
	}
}

func TestDupesConserveRequiresConfirmation(t *testing.T) {
	fakeBin := createFakeBurrow(t, `#!/bin/sh
echo '{"conserved":true}'
`)

	t.Setenv(envBurrowPath, fakeBin)
	exec, err := NewExecutor()
	if err != nil {
		t.Fatal(err)
	}

	handler := handleDupesConserve(exec)

	// Call without confirmed — should be rejected.
	req := newCallToolRequest("burrow_dupes_conserve", map[string]any{
		"conserve_dir": "/tmp/conserve",
		"path":         "/tmp/scan",
	})
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result when confirmed is missing")
	}

	// Call with confirmed=true — should succeed.
	req = newCallToolRequest("burrow_dupes_conserve", map[string]any{
		"conserve_dir": "/tmp/conserve",
		"path":         "/tmp/scan",
		"confirmed":    true,
	})
	result, err = handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Errorf("expected success when confirmed=true, got error: %v", result.Content)
	}
}

func TestReadOnlyHandlers(t *testing.T) {
	fakeBin := createFakeBurrow(t, `#!/bin/sh
echo '{"ok":true}'
`)

	t.Setenv(envBurrowPath, fakeBin)
	exec, err := NewExecutor()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		handler func(*Executor) server.ToolHandlerFunc
		args    map[string]any
	}{
		{"status", handleStatus, nil},
		{"analyze", handleAnalyze, map[string]any{"path": "/tmp"}},
		{"dupes", handleDupes, map[string]any{"path": "/tmp"}},
		{"doctor", handleDoctor, nil},
		{"size", handleSize, nil},
		{"report", handleReport, nil},
		{"clean_preview", handleCleanPreview, nil},
		{"dupes_restore", handleDupesRestore, map[string]any{"conserve_dir": "/tmp/c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.handler(exec)
			args := tt.args
			if args == nil {
				args = map[string]any{}
			}
			req := newCallToolRequest("burrow_"+tt.name, args)
			result, err := handler(context.Background(), req)
			if err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if result.IsError {
				t.Errorf("expected success, got error: %v", result.Content)
			}
		})
	}
}

func TestRunCommandNonZeroExit(t *testing.T) {
	fakeBin := createFakeBurrow(t, `#!/bin/sh
echo "error details" >&2
exit 2
`)

	t.Setenv(envBurrowPath, fakeBin)
	exec, err := NewExecutor()
	if err != nil {
		t.Fatal(err)
	}

	result, err := runCommand(context.Background(), exec, "status", "--json")
	if err != nil {
		t.Fatalf("runCommand should not return Go error: %v", err)
	}
	if !result.IsError {
		t.Error("expected MCP error result for nonzero exit")
	}
}

// --- Test helpers ---

// createFakeBurrow writes a shell script to a temp dir and returns its path.
func createFakeBurrow(t *testing.T, script string) string {
	t.Helper()
	tmpDir := t.TempDir()
	fakeBin := filepath.Join(tmpDir, "burrow")
	if err := os.WriteFile(fakeBin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return fakeBin
}

// newCallToolRequest constructs a CallToolRequest for testing.
func newCallToolRequest(name string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

// newTestServer creates an MCPServer with all tools registered using a dummy executor.
func newTestServer() *server.MCPServer {
	srv := server.NewMCPServer(serverName, serverVersion,
		server.WithToolCapabilities(false),
	)

	// Use a no-op executor for schema-only tests.
	exec := &Executor{binaryPath: "/nonexistent/burrow"}
	registerTools(srv, exec)
	return srv
}
