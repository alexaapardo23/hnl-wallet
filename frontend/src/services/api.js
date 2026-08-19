const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

/** Thrown for any non-2xx response, carrying the API's own error message. */
export class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
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
    throw new ApiError(data.error || "Ocurrió un error inesperado.", response.status);
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
