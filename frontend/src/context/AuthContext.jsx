import { createContext, useCallback, useEffect, useState } from "react";
import { authService } from "../services/api";

const STORAGE_KEY = "hnl_wallet_token";

export const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => localStorage.getItem(STORAGE_KEY));
  const [user, setUser] = useState(null);
  // True only while restoring a session found in localStorage on first
  // load — the initial GET /me confirms the token is still valid.
  const [loading, setLoading] = useState(() => Boolean(localStorage.getItem(STORAGE_KEY)));

  useEffect(() => {
    if (!token) {
      setLoading(false);
      return;
    }

    authService
      .me(token)
      .then(setUser)
      .catch(() => {
        // Token expired or invalid — drop the stale session.
        localStorage.removeItem(STORAGE_KEY);
        setToken(null);
        setUser(null);
      })
      .finally(() => setLoading(false));
    // Only re-run if the token itself changes (e.g. a fresh login).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const login = useCallback(async (email, password, accountNumber) => {
    const data = await authService.login(email, password, accountNumber);
    localStorage.setItem(STORAGE_KEY, data.token);
    setUser(data.user);
    setToken(data.token);
    return data;
  }, []);

  // Registration opens the user's first account server-side (see
  // README "Crear cuenta bancaria al registrar usuario") and returns the
  // same {token, user} shape as login, so this is login's twin rather than
  // a separate flow the caller has to chain into a login call itself.
  const register = useCallback(async (email, password, fullName, accountType) => {
    const data = await authService.register(email, password, fullName, accountType);
    localStorage.setItem(STORAGE_KEY, data.token);
    setUser(data.user);
    setToken(data.token);
    return data;
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY);
    setToken(null);
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ token, user, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
