import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useAccountDetail } from "../hooks/useAccountDetail";
import { OperationModal } from "../components/OperationModal";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import "./AccountDetail.css";

const ACCOUNT_TYPE_LABELS = {
  checking: "Checking",
  savings: "Savings",
  investment: "Investment",
};

const TRANSACTION_TYPE_LABELS = {
  deposit: "Deposit",
  withdrawal: "Withdrawal",
  transfer: "Transfer",
  internal_transfer: "Internal Transfer",
  initial_balance: "Initial Balance",
};

const currencyFormatter = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  signDisplay: "never",
});

const dateFormatter = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
});

function maskAccountNumber(accountNumber) {
  return `•••• ${accountNumber.slice(-4)}`;
}

// There's no free-text memo on a transaction — TigerBeetle transfers don't
// carry one (see README Historical Transaction Import) — so "description"
// is built entirely from fields the API already returns: type, direction,
// and counterparty_account_number. Never fabricated placeholder text.
function describeTransaction(tx) {
  const counterparty =
    tx.counterparty_account_number === "EXTERNAL"
      ? "external"
      : maskAccountNumber(tx.counterparty_account_number);

  switch (tx.type) {
    case "deposit":
      return `From ${counterparty}`;
    case "withdrawal":
      return `To ${counterparty}`;
    case "transfer":
      return tx.direction === "incoming" ? `From ${counterparty}` : `To ${counterparty}`;
    case "internal_transfer":
      return tx.direction === "incoming"
        ? `From your ${counterparty}`
        : `To your ${counterparty}`;
    case "initial_balance":
      return "Initial funding";
    default:
      return counterparty;
  }
}

export function AccountDetail() {
  const { accountNumber } = useParams();
  const { account, transactions, loading, error, reload } = useAccountDetail(accountNumber);
  const [activeOperation, setActiveOperation] = useState(null);

  return (
    <div className="account-detail-page">
      <header className="account-detail-header">
        <Link to="/dashboard" className="back-link">
          ← HNL Wallet
        </Link>
      </header>

      <main className="account-detail-content">
        {error && (
          <Alert>
            {error}{" "}
            <button className="retry-link" type="button" onClick={reload}>
              Reintentar
            </button>
          </Alert>
        )}

        {loading && <div className="account-summary-skeleton" aria-label="Cargando cuenta" />}

        {!loading && account && (
          <section className="account-summary">
            <p className="account-summary-type">
              {ACCOUNT_TYPE_LABELS[account.account_type] ?? account.account_type}
            </p>
            <p className="account-summary-number">{maskAccountNumber(account.account_number)}</p>
            <p className="account-summary-balance">
              {currencyFormatter.format(account.balance)} {account.currency}
            </p>

            <div className="account-actions">
              <Button type="button" onClick={() => setActiveOperation("deposit")}>
                Deposit
              </Button>
              <Button type="button" onClick={() => setActiveOperation("withdraw")}>
                Withdraw
              </Button>
              <Button type="button" onClick={() => setActiveOperation("transfer")}>
                Transfer
              </Button>
            </div>
          </section>
        )}

        <section className="transactions-section">
          <h2>Recent transactions</h2>

          <div className="transactions-table-wrap">
            <table className="transactions-table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Type</th>
                  <th>Description</th>
                  <th className="col-amount">Amount</th>
                </tr>
              </thead>
              <tbody>
                {loading &&
                  [0, 1, 2].map((i) => (
                    <tr key={i} className="transaction-row--skeleton">
                      <td colSpan={4} />
                    </tr>
                  ))}

                {!loading &&
                  transactions &&
                  transactions.map((tx) => (
                    <tr key={tx.id}>
                      <td className="col-date">{dateFormatter.format(new Date(tx.timestamp))}</td>
                      <td>{TRANSACTION_TYPE_LABELS[tx.type] ?? tx.type}</td>
                      <td className="col-description">{describeTransaction(tx)}</td>
                      <td
                        className={`col-amount transaction-amount transaction-amount--${tx.direction}`}
                      >
                        {tx.direction === "incoming" ? "+" : "-"}
                        {currencyFormatter.format(tx.amount)}
                      </td>
                    </tr>
                  ))}

                {!loading && transactions && transactions.length === 0 && (
                  <tr>
                    <td colSpan={4} className="transactions-empty">
                      Sin transacciones todavía.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>
      </main>

      {activeOperation && (
        <OperationModal
          type={activeOperation}
          accountNumber={accountNumber}
          onClose={() => setActiveOperation(null)}
          onSuccess={() => {
            setActiveOperation(null);
            reload();
          }}
        />
      )}
    </div>
  );
}
