import { Navigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import "./ui.css";

export function ProtectedRoute({ children }) {
  const { token, loading } = useAuth();

  if (loading) {
    return (
      <div style={{ display: "flex", justifyContent: "center", padding: 48 }}>
        <span className="spinner spinner--muted" aria-label="Cargando" />
      </div>
    );
  }

  if (!token) return <Navigate to="/login" replace />;

  return children;
}
