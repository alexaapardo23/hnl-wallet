import "./ui.css";

export function TextField({ id, label, ...rest }) {
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <input id={id} name={id} {...rest} />
    </div>
  );
}
