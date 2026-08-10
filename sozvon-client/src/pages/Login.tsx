//sozvon-client\src\pages\Login.tsx
import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { login as loginRequest } from "../api/auth";
import "../styles/Auth.css";

export default function Login() {
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const navigate = useNavigate();

  async function handleLogin() {
    try {
      const res = await loginRequest(login, password);
      localStorage.setItem("token", res.token);
      navigate("/app", { replace: true });
    } catch (e: any) {
      setError(e.message);
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h2>Login</h2>

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

        {error && <div className="auth-error">{error}</div>}

        <div className="auth-actions">
          <Link className="auth-link" to="/register">
            Register
          </Link>

          <button className="auth-button" onClick={handleLogin}>
            Login
          </button>
        </div>
      </div>
    </div>
  );
}
