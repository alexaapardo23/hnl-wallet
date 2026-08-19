const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

/** Thrown for any non-2xx response, carrying the API's own error message. */
export class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

// The API's error strings are plain English (see backend/api/*.go) — the
// frontend is Spanish throughout, so every message it can actually return
// is translated here. Anything not in this list (a bug, or a message added
// to the API later) falls back to the original English text rather than
// hiding it, so it stays visible and debuggable instead of silently wrong.
const ERROR_TRANSLATIONS = {
  "invalid email or password": "Email o contraseña incorrectos.",
  "multiple accounts share this email; account_number is required to log in":
    "Varias cuentas comparten este email. Se necesita el número de cuenta para iniciar sesión.",
  "missing bearer token": "Tu sesión no es válida. Inicia sesión de nuevo.",
  "invalid or expired token": "Tu sesión expiró. Inicia sesión de nuevo.",
  "account not found": "Cuenta no encontrada.",
  "from_account not found": "La cuenta de origen no existe o no te pertenece.",
  "to_account not found": "La cuenta destino no existe.",
  "from_account and to_account must be different": "La cuenta de origen y destino deben ser diferentes.",
  "from_account and to_account are required": "Debes indicar la cuenta de origen y destino.",
  "amount must be greater than 0": "El monto debe ser mayor a 0.",
  "insufficient funds": "Fondos insuficientes.",
  "email is already registered": "Este email ya está registrado.",
  "a valid email is required": "Ingresa un email válido.",
  "password must be at least 8 characters": "La contraseña debe tener al menos 8 caracteres.",
  "full_name is required": "El nombre completo es obligatorio.",
  "account_type must be one of: checking, savings, investment":
    "El tipo de cuenta debe ser checking, savings o investment.",
  "message is required": "Escribe un mensaje.",
  "confirmation_token is required": "Falta el token de confirmación.",
  "invalid or expired confirmation_token": "El token de confirmación no es válido o expiró.",
  "this confirmation_token was not issued to you": "Este token de confirmación no te pertenece.",
  "chat is not configured (OPENROUTER_API_KEY is not set)": "El chat no está disponible en este momento.",
};

// Some server errors embed a dynamic Go error (e.g. "failed to reach MCP
// server: dial tcp ...") that can't be matched as an exact string — those
// fall back to a translation by HTTP status instead of by literal text.
const STATUS_FALLBACKS = {
  502: "No se pudo conectar con el asistente. Intenta de nuevo.",
  503: "Este servicio no está disponible en este momento.",
};

function translateError(message, status) {
  return ERROR_TRANSLATIONS[message] ?? STATUS_FALLBACKS[status] ?? message;
}

async function request(path, { method = "GET", body, token } = {}) {
  const headers = { "Content-Type": "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;

  let response;
  try {
    response = await fetch(`${API_URL}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new ApiError("No se pudo conectar con el servidor. Verifica tu conexión.", 0);
  }

  const data = await response.json().catch(() => ({}));

  if (!response.ok) {
    throw new ApiError(
      translateError(data.error, response.status) || "Ocurrió un error inesperado.",
      response.status,
    );
  }

  return data;
}

export const authService = {
  /** @returns {Promise<import('../types').LoginResponse>} */
  login: (email, password, accountNumber) =>
    request("/auth/login", {
      method: "POST",
      body: accountNumber
        ? { email, password, account_number: accountNumber }
        : { email, password },
    }),

  /** @returns {Promise<import('../types').User>} */
  me: (token) => request("/me", { token }),
};

export const accountsService = {
  /**
   * Every balance here — per account and the total — is computed
   * server-side from TigerBeetle; this just relays what the API returns.
   * @returns {Promise<import('../types').AccountsSummary>}
   */
  summary: (token) => request("/accounts/summary", { token }),

  /** @returns {Promise<import('../types').AccountDetail>} */
  detail: (token, accountNumber) => request(`/accounts/${accountNumber}`, { token }),

  /** @returns {Promise<import('../types').Transaction[]>} */
  transactions: (token, accountNumber, limit = 10) =>
    request(`/accounts/${accountNumber}/transactions?limit=${limit}`, { token }),

  deposit: (token, accountNumber, amount) =>
    request(`/accounts/${accountNumber}/deposit`, { method: "POST", token, body: { amount } }),

  withdraw: (token, accountNumber, amount) =>
    request(`/accounts/${accountNumber}/withdraw`, { method: "POST", token, body: { amount } }),

  transfer: (token, fromAccount, toAccount, amount) =>
    request("/transfers", {
      method: "POST",
      token,
      body: { from_account: fromAccount, to_account: toAccount, amount },
    }),
};

export const chatService = {
  /**
   * React Chat -> POST /chat -> Go API -> OpenRouter -> tool call ->
   * MCP Server -> Go API -> TigerBeetle/PostgreSQL -> ... -> React.
   * If the model asked for deposit/withdraw/transfer, the response carries
   * requires_confirmation + confirmation_token instead of executing it —
   * see chatService.confirm.
   * @returns {Promise<import('../types').ChatResponse>}
   */
  send: (token, message) => request("/chat", { method: "POST", token, body: { message } }),

  /**
   * Executes a financial action POST /chat proposed, using the exact
   * amount/accounts signed into confirmationToken — nothing about the
   * pending action can be altered from here.
   * @returns {Promise<import('../types').ChatResponse>}
   */
  confirm: (token, confirmationToken) =>
    request("/chat/confirm", {
      method: "POST",
      token,
      body: { confirmation_token: confirmationToken },
    }),
};
