import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useChat } from "../hooks/useChat";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import "./Chat.css";

const SUGGESTIONS = [
  "¿Cuánto dinero tengo?",
  "¿Cuáles fueron mis últimas transacciones?",
  "Deposita $50 en mi cuenta",
];

export function Chat() {
  const { messages, sending, error, sendMessage, confirmAction } = useChat();
  const [draft, setDraft] = useState("");
  const bottomRef = useRef(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, sending]);

  function handleSubmit(event) {
    event.preventDefault();
    if (!draft.trim()) return;
    sendMessage(draft);
    setDraft("");
  }

  return (
    <div className="chat-page">
      <header className="chat-header">
        <Link to="/dashboard" className="back-link">
          ← HNL Wallet
        </Link>
        <h1>Asistente</h1>
      </header>

      <main className="chat-content">
        {messages.length === 0 && (
          <div className="chat-empty">
            <p>Pregúntame sobre tu dinero: balances, movimientos, o pídeme que hagas una operación.</p>
            <div className="chat-suggestions">
              {SUGGESTIONS.map((suggestion) => (
                <button
                  key={suggestion}
                  type="button"
                  className="chat-suggestion"
                  onClick={() => sendMessage(suggestion)}
                >
                  {suggestion}
                </button>
              ))}
            </div>
          </div>
        )}

        <ul className="chat-messages">
          {messages.map((message) => (
            <li key={message.id} className={`chat-message chat-message--${message.role}`}>
              <div className="chat-bubble">{message.content}</div>

              {message.pending && (
                <div className="chat-confirmation">
                  {message.pending.resolved ? (
                    <span className="chat-confirmation-done">Operación enviada.</span>
                  ) : (
                    <Button
                      type="button"
                      onClick={() => confirmAction(message.id)}
                      loading={sending}
                    >
                      Confirmar
                    </Button>
                  )}
                </div>
              )}
            </li>
          ))}

          {sending && (
            <li className="chat-message chat-message--assistant">
              <div className="chat-bubble chat-bubble--typing">
                <span className="spinner spinner--muted" aria-label="Escribiendo" />
              </div>
            </li>
          )}
        </ul>

        <div ref={bottomRef} />
      </main>

      <footer className="chat-footer">
        {error && <Alert>{error}</Alert>}

        <form className="chat-form" onSubmit={handleSubmit}>
          <input
            type="text"
            className="chat-input"
            placeholder="Escribe un mensaje..."
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            disabled={sending}
          />
          <Button type="submit" loading={sending} disabled={!draft.trim()}>
            Enviar
          </Button>
        </form>
      </footer>
    </div>
  );
}
