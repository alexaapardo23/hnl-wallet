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

const dateFormatter = new Intl.DateTimeFormat("es-ES", {
  day: "2-digit",
  month: "short",
  hour: "2-digit",
  minute: "2-digit",
});

function maskAccountNumber(accountNumber) {
  return `•••• ${accountNumber.slice(-4)}`;
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

          {loading && (
            <ul className="transactions-list">
              {[0, 1, 2].map((i) => (
                <li key={i} className="transaction-row transaction-row--skeleton" />
              ))}
            </ul>
          )}

          {!loading && transactions && (
            <ul className="transactions-list">
              {transactions.map((tx) => (
                <li key={tx.id} className="transaction-row">
                  <div className="transaction-info">
                    <span className="transaction-type">
                      {TRANSACTION_TYPE_LABELS[tx.type] ?? tx.type}
                    </span>
                    <span className="transaction-date">{dateFormatter.format(new Date(tx.timestamp))}</span>
                  </div>
                  <span
                    className={`transaction-amount transaction-amount--${tx.direction}`}
                  >
                    {tx.direction === "incoming" ? "+" : "-"}
                    {currencyFormatter.format(tx.amount)}
                  </span>
                </li>
              ))}

              {transactions.length === 0 && (
                <li className="transactions-empty">Sin transacciones todavía.</li>
              )}
            </ul>
          )}
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
