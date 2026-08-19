import { useCallback, useEffect, useState } from "react";
import { accountsService } from "../services/api";
import { useAuth } from "./useAuth";

/**
 * Fetches GET /accounts/summary — every balance in the result (per account
 * and the total) is already computed server-side from TigerBeetle; this
 * hook never adds numbers together itself, only stores and re-exposes what
 * the API returned.
 */
export function useAccountsSummary() {
  const { token } = useAuth();
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const reload = useCallback(() => {
    if (!token) return;

    setLoading(true);
    setError("");

    accountsService
      .summary(token)
      .then(setData)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  useEffect(reload, [reload]);

  return { data, loading, error, reload };
}
