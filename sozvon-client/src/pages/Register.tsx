//sozvon-client\src\pages\Register.tsx
import { useState } from "react";
import { Link } from "react-router-dom";
import { register as registerRequest } from "../api/auth";
import "../styles/Auth.css";

export default function Register() {
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  async function handleRegister() {
    setMessage("");
    setError("");

    if (!login || !password) {
      setError("Login and password are required");
      return;
    }

    try {
      await registerRequest(login, password);
      setMessage("Registration successful. You can now log in.");
      setLogin("");
      setPassword("");
    } catch (e: any) {
      setError(e.message || "Registration failed");
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h2>Registration</h2>

        <input
          className="auth-input"
          placeholder="Login"
          value={login}
          onChange={(e) => setLogin(e.target.value)}
        />

        <input
          className="auth-input"
          type="password"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />

        {message && <div className="auth-success">{message}</div>}
        {error && <div className="auth-error">{error}</div>}

        <div className="auth-actions">
          <Link className="auth-link" to="/login">
            Back to login
          </Link>

          <button className="auth-button" onClick={handleRegister}>
            Register
          </button>
        </div>
      </div>
    </div>
  );
}
