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
};

function translateError(message) {
  return ERROR_TRANSLATIONS[message] ?? message;
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
    throw new ApiError(translateError(data.error) || "Ocurrió un error inesperado.", response.status);
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
