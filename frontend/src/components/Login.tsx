import { useState } from 'react';
import type { FormEvent } from 'react';
import { api } from '../api/client';
import type { User } from '../types';

export function Login({ onLogin }: { onLogin: (user: User) => void }) {
  const [register, setRegister] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true); setError('');
    try {
      // Регистрация через API всегда создаёт роль user; назначение admin выполняется отдельно.
      if (register) await api.register(username, password);
      // Вход устанавливает HttpOnly cookie на сервере. В React передаётся пользователь, но не токен.
      onLogin(await api.login(username, password));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Не удалось войти');
    } finally { setBusy(false); }
  }

  return <main className="auth-page">
    <section className="auth-story"><span className="eyebrow">ЛИФТЫ / SIMULATOR</span>
      <h1>У каждого вызова<br />есть маршрут.</h1><p>Настрой здание, выбери стратегию и наблюдай за работой лифтов в реальном времени.</p>
      <div className="auth-shafts" aria-hidden="true"><i /><i /><i /></div>
      <small>До 20 этажей · До 5 лифтов · Две стратегии</small>
    </section>
    <form className="auth-card" onSubmit={submit}>
      <span className="eyebrow">СВОЙ СЕАНС СИМУЛЯЦИИ</span><h2>{register ? 'Создать аккаунт' : 'Войти в приложение'}</h2>
      <label>Логин<input autoComplete="username" value={username} onChange={e => setUsername(e.target.value)} required minLength={3} maxLength={40} placeholder="например, raman" /></label>
      <label>Пароль<input type="password" autoComplete={register ? 'new-password' : 'current-password'} value={password} onChange={e => setPassword(e.target.value)} required minLength={8} maxLength={72} /></label>
      {error && <div className="notice error" role="alert">{error}</div>}
      {/* Рисуем кнопку «Войти / Зарегистрироваться». */}
      <button className="primary" disabled={busy}>{busy ? 'Подождите…' : register ? 'Зарегистрироваться' : 'Войти'}</button>
      {/* Рисуем кнопку переключения входа и регистрации. */}
      <button type="button" className="text-button" onClick={() => { setRegister(!register); setError(''); }}>{register ? 'Уже есть аккаунт? Войти' : 'Нет аккаунта? Зарегистрироваться'}</button>
      <small>Новые пользователи получают обычную роль.</small>
    </form>
  </main>;
}
