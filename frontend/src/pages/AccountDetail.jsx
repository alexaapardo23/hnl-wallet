import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useAccountDetail } from "../hooks/useAccountDetail";
import { OperationModal } from "../components/OperationModal";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import "./AccountDetail.css";

const ACCOUNT_TYPE_LABELS = {
  checking: "Corriente",
  savings: "Ahorros",
  investment: "Inversión",
};

const TRANSACTION_TYPE_LABELS = {
  deposit: "Depósito",
  withdrawal: "Retiro",
  transfer: "Transferencia",
  internal_transfer: "Transferencia interna",
  initial_balance: "Saldo inicial",
};

const currencyFormatter = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  signDisplay: "never",
});

const dateFormatter = new Intl.DateTimeFormat("es-ES", {
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
      ? "el exterior"
      : maskAccountNumber(tx.counterparty_account_number);

  switch (tx.type) {
    case "deposit":
      return `Desde ${counterparty}`;
    case "withdrawal":
      return `Hacia ${counterparty}`;
    case "transfer":
      return tx.direction === "incoming" ? `Desde ${counterparty}` : `Hacia ${counterparty}`;
    case "internal_transfer":
      return tx.direction === "incoming"
        ? `Desde tu cuenta ${counterparty}`
        : `Hacia tu cuenta ${counterparty}`;
    case "initial_balance":
      return "Fondeo inicial";
    default:
      return counterparty;
  }
}

export function AccountDetail() {
  const { accountNumber } = useParams();
  const { account, transactions, loading, error, reload } = useAccountDetail(accountNumber);
  const [activeOperation, setActiveOperation] = useState(null);
  // Masked by default (matches the Dashboard list), but the user is
  // already looking at this one specific account on purpose here — e.g.
  // to read out or copy the full number for someone sending them a
  // transfer — so unlike the Dashboard, it needs to be revealable.
  const [numberRevealed, setNumberRevealed] = useState(false);

  // React Router reuses this component when navigating between two
  // /accounts/:accountNumber URLs (same route, different param) rather
  // than remounting it — without this, revealing one account's number and
  // then navigating to another would show the new one already revealed.
  useEffect(() => setNumberRevealed(false), [accountNumber]);

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
            <button
              type="button"
              className="account-summary-number"
              onClick={() => setNumberRevealed((prev) => !prev)}
              aria-label={numberRevealed ? "Ocultar número de cuenta" : "Ver número de cuenta completo"}
            >
              {numberRevealed ? account.account_number : maskAccountNumber(account.account_number)}
              <span className="account-summary-number-hint">
                {numberRevealed ? "Ocultar" : "Ver completo"}
              </span>
            </button>
            <p className="account-summary-balance">
              {currencyFormatter.format(account.balance)} {account.currency}
            </p>

            <div className="account-actions">
              <Button type="button" onClick={() => setActiveOperation("deposit")}>
                Depositar
              </Button>
              <Button type="button" onClick={() => setActiveOperation("withdraw")}>
                Retirar
              </Button>
              <Button type="button" onClick={() => setActiveOperation("transfer")}>
                Transferir
              </Button>
            </div>
          </section>
        )}

        <section className="transactions-section">
          <h2>Transacciones recientes</h2>

          <div className="transactions-table-wrap">
            <table className="transactions-table">
              <thead>
                <tr>
                  <th>Fecha</th>
                  <th>Tipo</th>
                  <th>Descripción</th>
                  <th className="col-amount">Monto</th>
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
