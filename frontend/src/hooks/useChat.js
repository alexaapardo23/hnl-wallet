import { useCallback, useState } from "react";
import { chatService } from "../services/api";
import { useAuth } from "./useAuth";

let nextId = 0;
function newId() {
  nextId += 1;
  return nextId;
}

/**
 * React Chat -> POST /chat -> Go API -> OpenRouter -> tool call ->
 * MCP Server -> Go API -> TigerBeetle/PostgreSQL -> ... -> React.
 *
 * A financial action (deposit/withdraw/transfer) never executes from
 * sendMessage's response alone — it comes back as a message with `pending`
 * set, and only confirmAction (POST /chat/confirm) can actually run it.
 */
export function useChat() {
  const { token } = useAuth();
  const [messages, setMessages] = useState([]);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState("");

  const sendMessage = useCallback(
    async (text) => {
      const trimmed = text.trim();
      if (!trimmed || sending) return;

      setError("");
      setMessages((prev) => [...prev, { id: newId(), role: "user", content: trimmed }]);
      setSending(true);

      try {
        const response = await chatService.send(token, trimmed);
        setMessages((prev) => [
          ...prev,
          {
            id: newId(),
            role: "assistant",
            content: response.reply,
            pending: response.requires_confirmation
              ? { confirmationToken: response.confirmation_token, resolved: false }
              : null,
          },
        ]);
      } catch (err) {
        setError(err.message);
      } finally {
        setSending(false);
      }
    },
    [token, sending],
  );

  const confirmAction = useCallback(
    async (messageId) => {
      const message = messages.find((m) => m.id === messageId);
      if (!message?.pending || message.pending.resolved || sending) return;

      setError("");
      setSending(true);

      try {
        const response = await chatService.confirm(token, message.pending.confirmationToken);
        setMessages((prev) => [
          ...prev.map((m) =>
            m.id === messageId ? { ...m, pending: { ...m.pending, resolved: true } } : m,
          ),
          { id: newId(), role: "assistant", content: response.reply },
        ]);
      } catch (err) {
        setError(err.message);
      } finally {
        setSending(false);
      }
    },
    [messages, token, sending],
  );

  return { messages, sending, error, sendMessage, confirmAction };
}
