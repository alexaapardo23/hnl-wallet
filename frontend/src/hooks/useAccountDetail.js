import { useCallback, useEffect, useState } from "react";
import { accountsService } from "../services/api";
import { useAuth } from "./useAuth";

/**
 * Fetches an account's detail (balance already computed server-side from
 * TigerBeetle) and its recent transactions together, and exposes a reload
 * so operations (deposit/withdraw/transfer) can refresh both after they
 * succeed.
 */
export function useAccountDetail(accountNumber) {
  const { token } = useAuth();
  const [account, setAccount] = useState(null);
  const [transactions, setTransactions] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const reload = useCallback(() => {
    if (!token || !accountNumber) return;

    setLoading(true);
    setError("");

    Promise.all([
      accountsService.detail(token, accountNumber),
      accountsService.transactions(token, accountNumber),
    ])
      .then(([accountData, transactionsData]) => {
        setAccount(accountData);
        setTransactions(transactionsData);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token, accountNumber]);

  useEffect(reload, [reload]);

  return { account, transactions, loading, error, reload };
}
