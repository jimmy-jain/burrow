package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerTools adds all burrow tool definitions and handlers to the MCP server.
func registerTools(srv *server.MCPServer, exec *Executor) {
	// Read-only tools.
	srv.AddTool(toolStatus(), handleStatus(exec))
	srv.AddTool(toolAnalyze(), handleAnalyze(exec))
	srv.AddTool(toolDupes(), handleDupes(exec))
	srv.AddTool(toolDoctor(), handleDoctor(exec))
	srv.AddTool(toolSize(), handleSize(exec))
	srv.AddTool(toolReport(), handleReport(exec))

	// Destructive tools.
	srv.AddTool(toolCleanPreview(), handleCleanPreview(exec))
	srv.AddTool(toolCleanExecute(), handleCleanExecute(exec))
	srv.AddTool(toolDupesConserve(), handleDupesConserve(exec))
	srv.AddTool(toolDupesRestore(), handleDupesRestore(exec))
}

// --- Read-only tool definitions ---

func toolStatus() mcp.Tool {
	return mcp.NewTool("burrow_status",
		mcp.WithDescription("System health metrics including CPU, memory, disk, network, and battery status"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

func toolAnalyze() mcp.Tool {
	return mcp.NewTool("burrow_analyze",
		mcp.WithDescription("Disk usage analysis for a given directory path"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("path",
			mcp.Description("Directory path to analyze"),
			mcp.Required(),
		),
	)
}

func toolDupes() mcp.Tool {
	return mcp.NewTool("burrow_dupes",
		mcp.WithDescription("Find duplicate files by content hash, reporting reclaimable space"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("path",
			mcp.Description("Directory path to scan for duplicates"),
			mcp.Required(),
		),
		mcp.WithString("min_size",
			mcp.Description("Minimum file size to consider (e.g. 1MB, 500KB)"),
			mcp.DefaultString("1MB"),
		),
	)
}

func toolDoctor() mcp.Tool {
	return mcp.NewTool("burrow_doctor",
		mcp.WithDescription("Developer environment health checks (Homebrew, Xcode, shell, git, etc.)"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

func toolSize() mcp.Tool {
	return mcp.NewTool("burrow_size",
		mcp.WithDescription("Developer cache sizes (Homebrew, npm, pip, Docker, etc.)"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

func toolReport() mcp.Tool {
	return mcp.NewTool("burrow_report",
		mcp.WithDescription("Full machine health snapshot combining status, size, and doctor output"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

// --- Destructive tool definitions ---

func toolCleanPreview() mcp.Tool {
	return mcp.NewTool("burrow_clean_preview",
		mcp.WithDescription("Preview what would be cleaned (dry-run, no files deleted)"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	)
}

func toolCleanExecute() mcp.Tool {
	return mcp.NewTool("burrow_clean_execute",
		mcp.WithDescription("Execute system cleanup — removes caches, logs, and temporary files"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithBoolean("confirmed",
			mcp.Description("Must be true to proceed with cleanup"),
			mcp.Required(),
		),
	)
}

func toolDupesConserve() mcp.Tool {
	return mcp.NewTool("burrow_dupes_conserve",
		mcp.WithDescription("Move duplicate files to a conservation directory with a restore manifest"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("conserve_dir",
			mcp.Description("Directory to move duplicate files into"),
			mcp.Required(),
		),
		mcp.WithString("path",
			mcp.Description("Directory path to scan for duplicates"),
			mcp.Required(),
		),
		mcp.WithBoolean("confirmed",
			mcp.Description("Must be true to proceed with moving files"),
			mcp.Required(),
		),
	)
}

func toolDupesRestore() mcp.Tool {
	return mcp.NewTool("burrow_dupes_restore",
		mcp.WithDescription("Restore previously conserved duplicate files from a conservation directory"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("conserve_dir",
			mcp.Description("Conservation directory to restore files from"),
			mcp.Required(),
		),
	)
}

// --- Read-only handlers ---

func handleStatus(exec *Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return runCommand(ctx, exec, "status", "--json")
	}
}

func handleAnalyze(exec *Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := request.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter: path"), nil
		}
		return runCommand(ctx, exec, "analyze", "--json", path)
	}
}

func handleDupes(exec *Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := request.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter: path"), nil
		}
		minSize := request.GetString("min_size", "1MB")
		return runCommand(ctx, exec, "dupes", "--json", "--min-size", minSize, path)
	}
}

func handleDoctor(exec *Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return runCommand(ctx, exec, "doctor", "--json")
	}
}

func handleSize(exec *Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return runCommand(ctx, exec, "size", "--json")
	}
}

func handleReport(exec *Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return runCommand(ctx, exec, "report")
	}
}

// --- Destructive handlers ---

func handleCleanPreview(exec *Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return runCommand(ctx, exec, "clean", "--dry-run")
	}
}

func handleCleanExecute(exec *Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		confirmed := request.GetBool("confirmed", false)
		if !confirmed {
			return mcp.NewToolResultError(ErrNotConfirmed.Error()), nil
		}
		return runCommand(ctx, exec, "clean")
	}
}

func handleDupesConserve(exec *Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		confirmed := request.GetBool("confirmed", false)
		if !confirmed {
			return mcp.NewToolResultError(ErrNotConfirmed.Error()), nil
		}
		conserveDir, err := request.RequireString("conserve_dir")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter: conserve_dir"), nil
		}
		path, err := request.RequireString("path")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter: path"), nil
		}
		return runCommand(ctx, exec, "dupes", "--conserve", conserveDir, path)
	}
}

func handleDupesRestore(exec *Executor) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		conserveDir, err := request.RequireString("conserve_dir")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter: conserve_dir"), nil
		}
		return runCommand(ctx, exec, "dupes", "--restore", conserveDir)
	}
}

// --- Shared command runner ---

// runCommand executes a burrow CLI command and returns the result as MCP tool output.
// On success, stdout is returned as text content. On failure, a structured error
// with stderr details is returned.
func runCommand(ctx context.Context, exec *Executor, args ...string) (*mcp.CallToolResult, error) {
	result, err := exec.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("command execution failed: %s", err)), nil
	}

	if result.ExitCode != 0 {
		errMsg := result.Stderr
		if errMsg == "" {
			errMsg = result.Stdout
		}
		return mcp.NewToolResultError(
			fmt.Sprintf("burrow %s failed (exit %d): %s", args[0], result.ExitCode, errMsg),
		), nil
	}

	output := result.Stdout
	if output == "" {
		output = result.Stderr
	}

	return mcp.NewToolResultText(output), nil
}
