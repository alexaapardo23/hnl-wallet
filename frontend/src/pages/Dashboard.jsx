import { Link } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import { useAccountsSummary } from "../hooks/useAccountsSummary";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import "./Dashboard.css";

const ACCOUNT_TYPE_LABELS = {
  checking: "Checking",
  savings: "Savings",
  investment: "Investment",
};

const currencyFormatter = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
});

function maskAccountNumber(accountNumber) {
  const lastFour = accountNumber.slice(-4);
  return `•••• ${lastFour}`;
}

export function Dashboard() {
  const { user, logout } = useAuth();
  const { data, loading, error, reload } = useAccountsSummary();

  const firstName = user?.full_name?.split(" ")[0] ?? "";

  return (
    <div className="dashboard-page">
      <header className="dashboard-header">
        <h1>HNL Wallet</h1>
        <div className="dashboard-header-right">
          <span className="dashboard-user">{firstName}</span>
          <Button type="button" onClick={logout}>
            Cerrar sesión
          </Button>
        </div>
      </header>

      <main className="dashboard-content">
        {error && (
          <Alert>
            {error}{" "}
            <button className="retry-link" type="button" onClick={reload}>
              Reintentar
            </button>
          </Alert>
        )}

        <section className="balance-card">
          <p className="balance-label">Total Balance</p>
          {loading ? (
            <div className="balance-skeleton" aria-label="Cargando balance" />
          ) : (
            <p className="balance-amount">
              {data ? currencyFormatter.format(data.total_balance) : "—"}
            </p>
          )}
        </section>

        <section className="accounts-section">
          <h2>My Accounts</h2>

          {loading && (
            <ul className="accounts-list">
              {[0, 1, 2].map((i) => (
                <li key={i} className="account-row account-row--skeleton" />
              ))}
            </ul>
          )}

          {!loading && data && (
            <ul className="accounts-list">
              {data.accounts.map((account) => (
                <li key={account.account_number}>
                  <Link to={`/accounts/${account.account_number}`} className="account-row">
                    <div className="account-info">
                      <span className="account-type">
                        {ACCOUNT_TYPE_LABELS[account.account_type] ?? account.account_type}
                      </span>
                      <span className="account-number">
                        {maskAccountNumber(account.account_number)}
                      </span>
                    </div>
                    <span className="account-balance">
                      {currencyFormatter.format(account.balance)}
                    </span>
                  </Link>
                </li>
              ))}

              {data.accounts.length === 0 && (
                <li className="accounts-empty">Todavía no tienes cuentas.</li>
              )}
            </ul>
          )}
        </section>
      </main>
    </div>
  );
}
