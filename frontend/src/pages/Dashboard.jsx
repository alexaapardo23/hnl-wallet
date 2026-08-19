import { useAuth } from "../hooks/useAuth";
import { Button } from "../components/Button";
import "./Dashboard.css";

// Minimal placeholder confirming the login flow lands somewhere real and
// the session works — the actual accounts/balances dashboard is a separate
// piece of work.
export function Dashboard() {
  const { user, logout } = useAuth();

  return (
    <div className="dashboard-page">
      <header className="dashboard-header">
        <h1>HNL Wallet</h1>
        <Button type="button" onClick={logout}>
          Cerrar sesión
        </Button>
      </header>

      <main className="dashboard-content">
        <h2>Bienvenido, {user?.full_name ?? "..."}</h2>
        <p>{user?.email}</p>
      </main>
    </div>
  );
}
