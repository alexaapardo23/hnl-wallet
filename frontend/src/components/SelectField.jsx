import "./ui.css";

export function SelectField({ id, label, children, ...rest }) {
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <select id={id} name={id} {...rest}>
        {children}
      </select>
    </div>
  );
}
