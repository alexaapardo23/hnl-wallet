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

	"hnl-wallet/backend/auth"
	"hnl-wallet/backend/openrouter"
)

const chatSystemPrompt = `You are the HNL Wallet assistant. You can see the authenticated user's own accounts, balances, and transaction history, and nothing else, through the tools available to you. Always call a tool to check real data before answering a question about money — never guess or estimate a balance.

When the user asks to deposit, withdraw, or transfer money: immediately call the matching tool (deposit, withdraw, or transfer) with the exact amount and account(s) from their message. Do this on your very first response — do not ask the user to confirm in your own words first, and do not describe what you are about to do instead of calling the tool. Confirmation is handled automatically by the system after your tool call, using its own message to the user, not yours — calling the tool never moves money by itself. If any required detail (amount, account) is genuinely missing from the user's message, ask for it before calling the tool; otherwise, call it right away.

Always reply in Spanish, regardless of what language the user writes in — the HNL Wallet frontend is Spanish-only, so a reply in any other language would look broken to every user. All amounts are in USD.`

// financialActionTools names every tool that moves money. POST /chat never
// invokes these directly on a model's tool call, no matter what the model
// (or a user's prompt) asks for — see chatHandler and README Financial
// Actions Require Confirmation.
var financialActionTools = map[string]bool{
	"deposit":  true,
	"withdraw": true,
	"transfer": true,
}

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Reply string `json:"reply"`

	// Set only when the model requested a financial action. No money has
	// moved yet — the caller must POST /chat/confirm with ConfirmationToken
	// to actually execute it.
	RequiresConfirmation bool           `json:"requires_confirmation,omitempty"`
	ConfirmationToken    string         `json:"confirmation_token,omitempty"`
	PendingAction        *pendingAction `json:"pending_action,omitempty"`
}

type pendingAction struct {
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

// chatHandler implements:
//
//	Usuario -> Chat -> OpenRouter/LLM -> tool call -> MCP -> HNL Wallet API -> TigerBeetle -> resultado -> LLM -> respuesta
//
// The user's own bearer token is forwarded, unmodified, all the way to the
// MCP server (see cmd/mcp-server) and from there to the HNL Wallet API —
// this handler never bypasses the ownership checks every other endpoint
// enforces. Concretely: if the model is asked to "transfer money from
// Pedro's account" and complies by calling transfer with someone else's
// from_account, this handler still forwards only the caller's own JWT: the
// HNL Wallet API's ownership check (not this handler, not the model, not
// MCP) is what actually rejects it. See README "MCP does not replace our
// authorization system" for a worked example.
//
// Read tools (get_accounts, get_balance, get_transaction_history) execute
// immediately, same as before. Financial action tools (deposit, withdraw,
// transfer) never do — this handler intercepts them before they ever reach
// MCP and returns a signed confirmation token instead; only
// chatConfirmHandler, given that exact token back, executes them.
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

	userID := userIDFromContext(r)
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

	// A financial action always wins over any read tool call in the same
	// turn: stop and ask for confirmation before running anything else.
	for _, call := range reply.ToolCalls {
		if !financialActionTools[call.Function.Name] {
			continue
		}

		resp, err := s.buildConfirmationResponse(userID, call)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to prepare confirmation: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, resp)
		return
	}

	// One round of tool execution for read tools: get_accounts, get_balance,
	// and get_transaction_history each answer their question in a single
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

// buildConfirmationResponse signs the proposed action (tool + its exact
// arguments) into a short-lived token bound to userID, and writes a
// deterministic, code-authored summary — not an LLM-generated one — so the
// text shown to the user is guaranteed to match what will actually execute
// if they confirm.
func (s *Server) buildConfirmationResponse(userID string, call openrouter.ToolCall) (chatResponse, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return chatResponse{}, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	summary, err := describeFinancialAction(call.Function.Name, args)
	if err != nil {
		return chatResponse{}, err
	}

	token, err := auth.GenerateConfirmationToken(s.JWTSecret, userID, call.Function.Name, call.Function.Arguments)
	if err != nil {
		return chatResponse{}, fmt.Errorf("failed to sign confirmation token: %w", err)
	}

	return chatResponse{
		Reply:                fmt.Sprintf("%s ¿Confirmas esta operación?", summary),
		RequiresConfirmation: true,
		ConfirmationToken:    token,
		PendingAction:        &pendingAction{Tool: call.Function.Name, Arguments: args},
	}, nil
}

func describeFinancialAction(tool string, args map[string]any) (string, error) {
	amount, _ := args["amount"].(float64)

	switch tool {
	case "deposit":
		return fmt.Sprintf("Vas a depositar $%.2f USD en la cuenta %v.", amount, args["account_number"]), nil
	case "withdraw":
		return fmt.Sprintf("Vas a retirar $%.2f USD de la cuenta %v.", amount, args["account_number"]), nil
	case "transfer":
		return fmt.Sprintf("Vas a transferir $%.2f USD desde %v hacia %v.", amount, args["from_account"], args["to_account"]), nil
	default:
		return "", fmt.Errorf("unknown financial action tool: %q", tool)
	}
}

type chatConfirmRequest struct {
	ConfirmationToken string `json:"confirmation_token"`
}

// chatConfirmHandler executes a financial action previously proposed by
// chatHandler, and only that exact action: the tool name and arguments come
// from the signed token, never from this request body, so nothing about the
// pending action can change between proposal and execution. The token is
// also checked against the caller's own user_id — a token is only usable by
// the user it was issued to, even though it's already scoped to whatever
// that user's JWT allows.
func (s *Server) chatConfirmHandler(w http.ResponseWriter, r *http.Request) {
	if s.OpenRouter == nil {
		writeError(w, http.StatusServiceUnavailable, "chat is not configured (OPENROUTER_API_KEY is not set)")
		return
	}

	var req chatConfirmRequest
	if err := decodeJSON(r, &req); err != nil || req.ConfirmationToken == "" {
		writeError(w, http.StatusBadRequest, "confirmation_token is required")
		return
	}

	claims, err := auth.ParseConfirmationToken(s.JWTSecret, req.ConfirmationToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired confirmation_token")
		return
	}
	if claims.UserID != userIDFromContext(r) {
		writeError(w, http.StatusForbidden, "this confirmation_token was not issued to you")
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

	result, err := session.client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      claims.Tool,
			Arguments: json.RawMessage(claims.Arguments),
		},
	})

	var resultText string
	switch {
	case err != nil:
		resultText = fmt.Sprintf("error calling tool: %v", err)
	default:
		resultText = toolResultText(result)
	}

	// Ask the model to phrase the outcome as a plain fact to relay, rather
	// than reconstructing a fake tool-call turn (fragile with smaller
	// models) — this is the one and only time this specific action
	// executes, regardless of how the reply ends up phrasing it.
	messages := []openrouter.Message{
		{Role: "system", Content: "You relay the outcome of a financial action the user already confirmed and that has now just been attempted. Summarize the result below in one or two short sentences in Spanish, clearly stating whether it succeeded or failed and why. Do not call any tool and do not ask any question."},
		{Role: "user", Content: fmt.Sprintf("Result of the confirmed %s: %s", claims.Tool, resultText)},
	}

	reply, err := s.OpenRouter.ChatCompletion(ctx, messages, nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("OpenRouter request failed: %v", err))
		return
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
