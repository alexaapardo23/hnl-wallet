import { useState } from "react";
import { Modal } from "./Modal";
import { TextField } from "./TextField";
import { Button } from "./Button";
import { Alert } from "./Alert";
import { accountsService } from "../services/api";
import { useAuth } from "../hooks/useAuth";

const TITLES = {
  deposit: "Deposit",
  withdraw: "Withdraw",
  transfer: "Transfer",
};

/**
 * One modal for deposit, withdraw, and transfer — same shape (amount, plus
 * a destination account for transfer), same submit/loading/error handling.
 * On success, calls onSuccess so the caller can reload the account and its
 * transactions (the new balance always comes back from the API response,
 * never computed here).
 */
export function OperationModal({ type, accountNumber, onClose, onSuccess }) {
  const { token } = useAuth();
  const [amount, setAmount] = useState("");
  const [toAccount, setToAccount] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event) {
    event.preventDefault();
    setError("");

    const numericAmount = Number(amount);
    if (!(numericAmount > 0)) {
      setError("El monto debe ser mayor a 0.");
      return;
    }
    if (type === "transfer" && !toAccount.trim()) {
      setError("Ingresa la cuenta destino.");
      return;
    }

    setLoading(true);
    try {
      if (type === "deposit") {
        await accountsService.deposit(token, accountNumber, numericAmount);
      } else if (type === "withdraw") {
        await accountsService.withdraw(token, accountNumber, numericAmount);
      } else {
        await accountsService.transfer(token, accountNumber, toAccount.trim(), numericAmount);
      }
      onSuccess();
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <Modal title={TITLES[type]} onClose={onClose}>
      <form className="operation-form" onSubmit={handleSubmit}>
        <Alert>{error}</Alert>

        {type === "transfer" && (
          <TextField
            id="to_account"
            label="Cuenta destino"
            placeholder="4001-0000-0000-0000"
            value={toAccount}
            onChange={(e) => setToAccount(e.target.value)}
            disabled={loading}
            required
          />
        )}

        <TextField
          id="amount"
          label="Monto (USD)"
          type="number"
          step="0.01"
          min="0.01"
          placeholder="0.00"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          disabled={loading}
          required
        />

        <Button type="submit" block loading={loading}>
          {loading ? "Procesando..." : TITLES[type]}
        </Button>
      </form>
    </Modal>
  );
}
