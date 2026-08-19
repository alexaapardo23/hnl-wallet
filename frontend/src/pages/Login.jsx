import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import { AMBIGUOUS_EMAIL_ERROR } from "../services/api";
import { Button } from "../components/Button";
import { TextField } from "../components/TextField";
import { Alert } from "../components/Alert";
import "./Login.css";

export function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [accountNumber, setAccountNumber] = useState("");
  // Revealed only after the API says this email alone doesn't identify a
  // single user — see README Login and Duplicate Emails. Most users never
  // see this field; it's only the seed's 20 duplicate-email accounts that
  // need it, and there was previously no way to fill it in at all, locking
  // those accounts out of the UI entirely even though the API supports
  // disambiguating them.
  const [needsAccountNumber, setNeedsAccountNumber] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event) {
    event.preventDefault();
    setError("");
    setLoading(true);

    try {
      // Login -> POST /auth/login -> JWT -> guardar sesión -> Dashboard
      await login(email, password, needsAccountNumber ? accountNumber : undefined);
      navigate("/dashboard");
    } catch (err) {
      if (err.original === AMBIGUOUS_EMAIL_ERROR) {
        setNeedsAccountNumber(true);
      }
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  function handleEmailChange(value) {
    setEmail(value);
    // A different email might not be ambiguous — don't keep asking for an
    // account_number that may no longer be relevant.
    setNeedsAccountNumber(false);
  }

  return (
    <div className="login-page">
      <form className="login-card" onSubmit={handleSubmit}>
        <div className="login-header">
          <h1>HNL Wallet</h1>
          <p>Inicia sesión en tu cuenta</p>
        </div>

        <Alert>{error}</Alert>

        <TextField
          id="email"
          label="Email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(e) => handleEmailChange(e.target.value)}
          disabled={loading}
          required
        />

        <TextField
          id="password"
          label="Contraseña"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          disabled={loading}
          required
        />

        {needsAccountNumber && (
          <TextField
            id="account_number"
            label="Número de cuenta"
            placeholder="4001-0000-0000-0000"
            value={accountNumber}
            onChange={(e) => setAccountNumber(e.target.value)}
            disabled={loading}
            required
          />
        )}

        <Button type="submit" block loading={loading}>
          {loading ? "Ingresando..." : "Iniciar sesión"}
        </Button>
      </form>
    </div>
  );
}
