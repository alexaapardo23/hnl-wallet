import "./ui.css";

export function Button({ children, loading, block, disabled, ...rest }) {
  return (
    <button
      className={`button${block ? " button--block" : ""}`}
      disabled={disabled || loading}
      {...rest}
    >
      {loading && <span className="spinner" aria-hidden="true" />}
      {children}
    </button>
  );
}
