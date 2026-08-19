import "./ui.css";

export function Alert({ children, variant = "danger" }) {
  if (!children) return null;
  return (
    <div className={`alert alert--${variant}`} role="alert">
      {children}
    </div>
  );
}
