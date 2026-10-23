package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// toolCallName extracts the tool name from a tools/call params object.
func toolCallName(params json.RawMessage) string {
	var call struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return ""
	}
	return call.Name
}

// toolCallArguments extracts the tool's own arguments from tools/call params.
// A missing or null value means no arguments, which handlers read as an empty
// object.
func toolCallArguments(params json.RawMessage) json.RawMessage {
	var call struct {
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return json.RawMessage(`{}`)
	}
	if len(call.Arguments) == 0 || string(call.Arguments) == "null" {
		return json.RawMessage(`{}`)
	}
	return call.Arguments
}

// listTools returns the complete Go-owned MCP tool registry.
func (h *handler) listTools() ([]json.RawMessage, error) {
	local := make([]json.RawMessage, 0, len(registry()))
	for _, t := range registry() {
		data, err := json.Marshal(map[string]any{
			"name":         t.Name,
			"title":        t.Title,
			"description":  t.Description,
			"inputSchema":  t.InputSchema,
			"outputSchema": t.OutputSchema,
			"annotations":  t.Annotations,
		})
		if err != nil {
			return nil, fmt.Errorf("encode local tool %s: %w", t.Name, err)
		}
		local = append(local, data)
	}

	return local, nil
}

// callTool enforces the plan before dispatch and contains handler failures.
func (h *handler) callTool(ctx context.Context, t *tool, params json.RawMessage, auth Auth) (result *callResult) {
	defer func() {
		if recovered := recover(); recovered != nil {
			h.deps.Logger.ErrorContext(ctx, "panic in MCP tool", "tool", t.Name,
				"panic_type", fmt.Sprintf("%T", recovered), "stack", string(debug.Stack()))
			result = errorResult("The tool could not finish. Please try again.")
		}
	}()
	planAccess := h.deps.Plans
	if planAccess == nil && h.deps.Billing != nil {
		planAccess = h.deps.Billing
	}
	if planAccess != nil {
		feature, known := toolFeature(t.Name)
		if !known {
			return errorResult("This tool is unavailable on your plan.")
		}
		if feature != "" {
			allowed, err := planAccess.Allows(ctx, auth.OrganizationID, feature)
			if err != nil {
				return errorResult("Plan access could not be checked. Please try again.")
			}
			if !allowed {
				return errorResult("This tool is not included in your plan.")
			}
		}
	}
	env := callEnv{h: h, deps: h.deps, auth: auth}
	result, err := t.Handler(ctx, params, &env)
	if err != nil {
		if message := spend.Message(err); message != "" {
			return errorResult(message)
		}
		if appErr, ok := errors.AsType[*appError](err); ok {
			return errorResult(appErr.Error())
		}
		h.deps.Logger.ErrorContext(ctx, "MCP tool failed", "tool", t.Name, "err", err)
		return errorResult("The tool could not finish. Please try again.")
	}
	if result == nil {
		return errorResult("The tool returned no result. Please try again.")
	}
	if auth.IsExternal() && auth.OrganizationID != "" && h.deps.DB != nil && !result.IsError {
		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		stamp := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := h.deps.DB.Exec(writeCtx, `INSERT INTO organization_activation_state
			(organization_id, first_mcp_authorized_at, first_mcp_tool_call_at, updated_at) VALUES ($1,$2,$2,$2)
			ON CONFLICT (organization_id) DO UPDATE SET
			first_mcp_authorized_at=COALESCE(organization_activation_state.first_mcp_authorized_at,EXCLUDED.first_mcp_authorized_at),
			first_mcp_tool_call_at=COALESCE(organization_activation_state.first_mcp_tool_call_at,EXCLUDED.first_mcp_tool_call_at), updated_at=EXCLUDED.updated_at`, auth.OrganizationID, stamp); err != nil {
			h.deps.Logger.WarnContext(ctx, "record MCP activation", "err", err)
		}
	}
	return result
}
