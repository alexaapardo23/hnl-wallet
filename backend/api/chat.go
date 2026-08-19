package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"hnl-wallet/backend/openrouter"
)

const chatSystemPrompt = `You are the HNL Wallet assistant. You can see the authenticated user's own accounts and balances, and nothing else, through the tools available to you. Always call a tool to check real data before answering a question about money — never guess or estimate a balance. Reply in the same language the user wrote in. All amounts are in USD.`

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Reply string `json:"reply"`
}

// chatHandler implements:
//
//	Usuario -> Chat -> OpenRouter/LLM -> tool call -> MCP -> HNL Wallet API -> TigerBeetle -> resultado -> LLM -> respuesta
//
// The user's own bearer token is forwarded, unmodified, all the way to the
// MCP server (see cmd/mcp-server) and from there to the HNL Wallet API —
// this handler never bypasses the ownership checks every other endpoint
// enforces; the LLM can only ever see what the user themselves could see
// by calling the API directly.
func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	if s.OpenRouter == nil {
		writeError(w, http.StatusServiceUnavailable, "chat is not configured (OPENROUTER_API_KEY is not set)")
		return
	}

	var req chatRequest
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	bearerToken := r.Header.Get("Authorization")
	ctx := r.Context()

	session, err := s.newMCPSession(ctx, bearerToken)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to reach MCP server: %v", err))
		return
	}
	defer session.client.Close()

	messages := []openrouter.Message{
		{Role: "system", Content: chatSystemPrompt},
		{Role: "user", Content: req.Message},
	}

	reply, err := s.OpenRouter.ChatCompletion(ctx, messages, session.tools)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("OpenRouter request failed: %v", err))
		return
	}
	messages = append(messages, reply)

	// One round of tool execution: every tool this first pass offers
	// (get_accounts, get_balance) answers a balance question in a single
	// call, so there's no need for a multi-step agent loop yet.
	for _, call := range reply.ToolCalls {
		result, err := session.client.CallTool(ctx, newCallToolRequest(call))

		var resultText string
		switch {
		case err != nil:
			resultText = fmt.Sprintf("error calling tool: %v", err)
		default:
			resultText = toolResultText(result)
		}

		messages = append(messages, openrouter.Message{
			Role:       "tool",
			ToolCallID: call.ID,
			Content:    resultText,
		})
	}

	if len(reply.ToolCalls) > 0 {
		reply, err = s.OpenRouter.ChatCompletion(ctx, messages, session.tools)
		if err != nil {
			writeError(w, http.StatusBadGateway, fmt.Sprintf("OpenRouter follow-up request failed: %v", err))
			return
		}
	}

	writeJSON(w, http.StatusOK, chatResponse{Reply: reply.Content})
}

// mcpSession bundles a live MCP client with the OpenRouter-formatted tool
// list fetched from it in the same call.
type mcpSession struct {
	client *mcpclient.Client
	tools  []openrouter.Tool
}

func newCallToolRequest(call openrouter.ToolCall) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      call.Function.Name,
			Arguments: json.RawMessage(call.Function.Arguments),
		},
	}
}

// newMCPSession connects to the MCP server, forwarding bearerToken as-is on
// every request (mcp-go re-sends configured headers per call, so this
// covers Initialize/ListTools/CallTool alike), and lists its tools in
// OpenRouter's function-calling format.
func (s *Server) newMCPSession(ctx context.Context, bearerToken string) (*mcpSession, error) {
	httpTransport, err := transport.NewStreamableHTTP(
		s.MCPServerURL,
		transport.WithHTTPHeaders(map[string]string{"Authorization": bearerToken}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP transport: %w", err)
	}

	client := mcpclient.NewClient(httpTransport)

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{Name: "hnl-wallet-chat", Version: "1.0.0"}

	if _, err := client.Initialize(ctx, initRequest); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize MCP session: %w", err)
	}

	toolsResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}

	tools := make([]openrouter.Tool, 0, len(toolsResult.Tools))
	for _, t := range toolsResult.Tools {
		schema, err := json.Marshal(t.InputSchema)
		if err != nil {
			continue
		}
		tools = append(tools, openrouter.Tool{
			Type: "function",
			Function: openrouter.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			},
		})
	}

	return &mcpSession{client: client, tools: tools}, nil
}

func toolResultText(result *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}
