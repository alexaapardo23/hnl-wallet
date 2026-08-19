import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import { Button } from "../components/Button";
import { TextField } from "../components/TextField";
import { SelectField } from "../components/SelectField";
import { Alert } from "../components/Alert";
import "./Login.css";

export function Register() {
  const { register } = useAuth();
  const navigate = useNavigate();

  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [accountType, setAccountType] = useState("checking");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event) {
    event.preventDefault();
    setError("");
    setLoading(true);

    try {
      // Register -> POST /auth/register (crea usuario + cuenta bancaria) ->
      // JWT -> guardar sesión -> Dashboard, igual que Login.
      await register(email, password, fullName, accountType);
      navigate("/dashboard");
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="login-page">
      <form className="login-card" onSubmit={handleSubmit}>
        <div className="login-header">
          <h1>HNL Wallet</h1>
          <p>Crea tu cuenta</p>
        </div>

        <Alert>{error}</Alert>

        <TextField
          id="full_name"
          label="Nombre completo"
          autoComplete="name"
          value={fullName}
          onChange={(e) => setFullName(e.target.value)}
          disabled={loading}
          required
        />

        <TextField
          id="email"
          label="Email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          disabled={loading}
          required
        />

        <TextField
          id="password"
          label="Contraseña"
          type="password"
          autoComplete="new-password"
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          disabled={loading}
          required
        />

        <SelectField
          id="account_type"
          label="Tipo de cuenta"
          value={accountType}
          onChange={(e) => setAccountType(e.target.value)}
          disabled={loading}
        >
          <option value="checking">Corriente</option>
          <option value="savings">Ahorros</option>
          <option value="investment">Inversión</option>
        </SelectField>

        <Button type="submit" block loading={loading}>
          {loading ? "Creando cuenta..." : "Crear cuenta"}
        </Button>

        <p className="login-footer">
          ¿Ya tienes cuenta? <Link to="/login">Inicia sesión</Link>
        </p>
      </form>
    </div>
  );
}
