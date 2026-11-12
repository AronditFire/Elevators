import { useCallback, useEffect, useState } from 'react';
import type { ChangeEvent } from 'react';
import type { Command, PeopleInput, Scenario, Session, SessionSummary, User } from './types';
import { api, download } from './api/client';
import { defaultConfig, exportDraft, parseConfig, parseScenario } from './lib/config';
import { useSession } from './hooks/useSession';
import { Login } from './components/Login';
import { ConfigEditor } from './components/ConfigEditor';
import { ScenarioEditor } from './components/ScenarioEditor';
import { Controls } from './components/Controls';
import { Building } from './components/Building';
import { PeoplePanel } from './components/PeoplePanel';
import { PeopleTable } from './components/PeopleTable';
import { StatisticsPanel } from './components/StatisticsPanel';
import { AdminPanel } from './components/AdminPanel';

const stateNames = { ready: 'Готов к старту', running: 'Работает', paused: 'На паузе', stopped: 'Остановлен' };
type Tab = 'simulation' | 'config' | 'scenario' | 'users';

export default function App() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [list, setList] = useState<SessionSummary[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>('config');
  const [config, setConfig] = useState(defaultConfig);
  const [scenario, setScenario] = useState<Scenario>({ events: [] });
  const [people, setPeople] = useState<PeopleInput>({ from: 1, to: 10, count: 1 });
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const [noticeError, setNoticeError] = useState(false);
  const { session, setSession, connected, error } = useSession(selected);

  const reportError = useCallback((e: unknown) => { setNotice(e instanceof Error ? e.message : 'Не удалось выполнить действие'); setNoticeError(true); }, []);
  // API списка сеансов учитывает владельца и роль; фильтрация доступа выполняется в Go.
  const refresh = useCallback(async () => { setList(await api.sessions()); }, []);
  // Повторный GET /auth/me учитывает изменение роли или блокировку другим администратором.
  const refreshUser = useCallback(() => { void api.me().then(setUser).catch(() => { setUser(null); setSelected(null); }); }, []);

  // При загрузке восстанавливаем пользователя через API, а не доверяем данным браузерного хранилища.
  useEffect(() => { void api.me().then(setUser).catch(() => setUser(null)).finally(() => setLoading(false)); }, []);
  useEffect(() => {
    if (!user) return;
    void refresh().catch(reportError);
    const timer = setInterval(() => { void refresh().catch(reportError); refreshUser(); }, 5000);
    return () => clearInterval(timer);
  }, [user?.id, user?.role, refresh, refreshUser, reportError]);
  useEffect(() => {
    // Черновики синхронизируем только при смене ID: частые WebSocket-снимки не должны затирать ввод.
    if (!session) return;
    setConfig(session.config); setScenario(session.scenario);
    setPeople({ from: 1, to: session.config.floors, count: 1 });
    setTab('simulation');
  }, [session?.id]);
  useEffect(() => { if (user?.role !== 'admin' && tab === 'users') setTab('simulation'); }, [user?.role, tab]);

  async function run(job: () => Promise<void>) {
    setBusy(true); setNotice('');
    try { await job(); } catch (e) { reportError(e); }
    finally { setBusy(false); }
  }
  function applied(current: Session) {
    setSession(current); setConfig(current.config); setScenario(current.scenario);
  }
  // Все действия пульта идут в Go API. Применяем подтверждённый снимок, без локального пересчёта модели.
  const command = (value: Command) => { if (selected) void run(async () => { setSession(await api.command(selected, value)); await refresh(); }); };

  async function upload(event: ChangeEvent<HTMLInputElement>, kind: 'config' | 'scenario') {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    await run(async () => {
      if (file.size > 1024 * 1024) throw new Error('YAML не больше 1 МБ');
      const text = await file.text();
      // Для созданного сеанса YAML сразу валидирует и применяет backend; без сеанса меняется только черновик.
      if (selected) applied(await api.upload(selected, kind, text));
      else if (kind === 'config') setConfig(parseConfig(text));
      else setScenario(parseScenario(text));
      setNotice('YAML загружен.'); setNoticeError(false);
    });
  }

  if (loading) return <main className="loading-page">Подключаемся к симулятору…</main>;
  if (!user) return <Login onLogin={setUser} />;
  const locked = session?.started ?? false;
  return <div className="app-shell">
    <aside className="sidebar"><a className="brand" href="#" onClick={e => { e.preventDefault(); setTab(session ? 'simulation' : 'config'); }}><span className="brand-icon">↕</span>ЛИФТЫ<small>SIMULATOR</small></a>
      <nav>
        {/* Рисуем кнопку «Симуляция». */}
        <button className={tab === 'simulation' ? 'current' : ''} disabled={!selected} onClick={() => setTab('simulation')}>▦ Симуляция</button>
        {/* Рисуем кнопку «Конфигурация». */}
        <button className={tab === 'config' ? 'current' : ''} onClick={() => setTab('config')}>⚙ Конфигурация</button>
        {/* Рисуем кнопку «Сценарий». */}
        <button className={tab === 'scenario' ? 'current' : ''} onClick={() => setTab('scenario')}>◷ Сценарий</button>{user.role === 'admin' && 
        /* Рисуем кнопку «Пользователи» для админа. */
        <button className={tab === 'users' ? 'current' : ''} onClick={() => setTab('users')}>♙ Пользователи</button>}</nav>
      <div className="session-picker"><span className="eyebrow">{user.role === 'admin' ? 'ДОСТУПНЫЕ СЕАНСЫ' : 'МОЙ СЕАНС'}</span>{list.map(item => 
        /* Рисуем кнопку выбора сеанса. */
        <button className={selected === item.id ? 'selected-session' : ''} key={item.id} onClick={() => { setSelected(item.id); setNotice(''); }}><strong>{item.owner_id === user.id ? 'Моя симуляция' : `Сеанс ${item.id.slice(0, 6)}`}</strong><small>{stateNames[item.state]} · {item.time.toFixed(1)} с</small></button>)}</div>
      {/* Рисуем кнопку «Новый сеанс». */}
      <button className="new-session" onClick={() => { setSelected(null); setConfig(defaultConfig()); setScenario({ events: [] }); setTab('config'); setNotice(''); }}>+ Новый сеанс</button>
      {/* Выход сначала отзывает токен через API, затем убирает данные пользователя из React-состояния. */}
      <div className="account"><strong>{user.username}</strong><small>{user.role === 'admin' ? 'Администратор' : 'Пользователь'}</small>
        {/* Рисуем кнопку «Выйти». */}
        <button className="text-button" disabled={busy} onClick={() => void run(async () => { await api.logout(); setSelected(null); setUser(null); setList([]); })}>Выйти →</button></div>
    </aside>
    <main className="workspace">
      <header className="page-header"><div><span className="eyebrow">СИСТЕМА КОНТРОЛЯ РАБОТЫ ЛИФТОВ</span><h1>{{ simulation: 'Симуляция здания', config: 'Конфигурация', scenario: 'Сценарий генерации', users: 'Пользователи' }[tab]}</h1></div><span className={`connection ${connected ? 'online' : ''}`}>{selected ? connected ? '● В реальном времени' : '○ Подключение…' : 'Новый сеанс'}</span></header>
      {notice && <div className={`notice ${noticeError ? 'error' : 'success'}`} role="status">{notice}
        {/* Рисуем кнопку закрытия сообщения. */}
        <button onClick={() => setNotice('')} aria-label="Закрыть сообщение">×</button></div>}
      {error && <div className="notice error" role="alert">{error}</div>}
      {selected && !session && tab === 'simulation' && <div className="card empty-state">Загружаем сеанс…</div>}
      {(tab === 'config' || tab === 'scenario') && <section className="card editor-card">
        <div className="section-line"><div><span className="eyebrow">{locked ? 'НАСТРОЙКИ ЗАФИКСИРОВАНЫ' : 'ДО ПЕРВОГО СТАРТА'}</span><h2>{tab === 'config' ? 'Здание и лифты' : 'Правила генерации'}</h2></div><span className="badge">{tab === 'config' ? `${config.elevators.length} лифта` : `${scenario.events.length} событий`}</span></div>
        {tab === 'config' ? <ConfigEditor value={config} onChange={setConfig} locked={locked || busy} /> : <ScenarioEditor value={scenario} onChange={setScenario} floors={config.floors} locked={locked || busy} />}
        <div className="editor-actions">
          {/* Рисуем кнопку «Загрузить YAML». */}
          <label className={`file-button ${locked || busy ? 'disabled' : ''}`}>Загрузить YAML ↑<input type="file" accept=".yaml,.yml" disabled={locked || busy} onChange={e => void upload(e, tab === 'config' ? 'config' : 'scenario')} /></label>
          {/* Из существующего сеанса API отдаёт применённый YAML; черновик экспортируется прямо в браузере. */}
          {/* Рисуем кнопку «Скачать YAML». */}
          <button className="secondary" disabled={busy} onClick={() => void run(async () => { const kind = tab === 'config' ? 'config' : 'scenario'; if (selected) await download(`/sessions/${selected}/${kind}.yaml`, `${kind}.yaml`); else exportDraft(tab === 'config' ? config : scenario, `${kind}.yaml`); })}>Скачать YAML ↓</button>
          {/* POST/PUT API создаёт сеанс или применяет настройки; сервер повторно проверяет запрет изменений после старта. */}
          {/* Рисуем кнопку «Создать сеанс / Применить настройки». */}
          <button className="primary" disabled={busy || locked} onClick={() => void run(async () => { const current = selected ? await api.configure(selected, config, scenario) : await api.create(config, scenario); setSelected(current.id); applied(current); setTab('simulation'); await refresh(); })}>{busy ? 'Применяем…' : selected ? 'Применить настройки' : 'Создать сеанс'}</button></div>
        {locked && <p className="muted">Для другой конфигурации останови и удали текущий сеанс, затем создай новый.</p>}
      </section>}
      {tab === 'simulation' && session && <>
        <div className="status-bar"><span className={`state-badge ${session.state}`}>{stateNames[session.state]}</span><span>Модельное время <strong>{session.time.toFixed(1)} с</strong></span><span>Доставлено <strong>{session.statistics.transported}</strong></span><span>В здании <strong>{session.people.length}</strong></span>{session.fire && <strong className="fire-label">ПОЖАРНАЯ ТРЕВОГА</strong>}</div>
        <Controls session={session} busy={busy} onCommand={command} />
        {session.last_event_error && <div className="notice error">{session.last_event_error}</div>}
        <div className="simulation-grid"><section className="card building-card"><div className="section-line"><h2>Здание</h2><span className="badge">{session.config.strategy === 'minimum_wait' ? 'Минимум ожидания' : 'Минимум холостых ходов'}</span></div><Building session={session} busy={busy} onCommand={command} onFloor={from => { setPeople(value => ({ ...value, from, to: value.to === from ? from === 1 ? session.config.floors : 1 : value.to })); document.getElementById('people-panel')?.scrollIntoView({ behavior: 'smooth', block: 'nearest' }); }} /></section>
          <section className="card" id="people-panel"><span className="eyebrow">ЭТАЖНЫЙ ПУЛЬТ</span><h2>Добавить людей</h2><PeoplePanel session={session} value={people} onChange={setPeople} busy={busy} onAdd={() => command({ type: 'people', people })} /><div className="scenario-progress"><small>Сценарий</small><strong>{session.event_index} / {session.scenario.events.length}</strong><p className="muted">событий выполнено</p></div></section></div>
        <section className="card"><div className="section-line"><h2>Люди в системе</h2><span className="badge">{session.people.length}</span></div><PeopleTable session={session} /></section>
        {/* CSV скачивается с backend после корректной остановки; показатели рассчитаны Go-ядром. */}
        {session.state === 'stopped' && <StatisticsPanel session={session} onExport={() => void run(async () => { await download(`/sessions/${session.id}/report.csv`, 'statistics.csv'); })} />}
        {/* Удаление выполняется через API; состояние интерфейса очищается только после успешного ответа. */}
        <div className="session-footer"><span>Сеанс {session.id.slice(0, 8)} · История не сохраняется</span>
          {/* Рисуем кнопку «Удалить сеанс». */}
          <button className="text-button" disabled={busy || !['ready', 'stopped'].includes(session.state)} onClick={() => void run(async () => { await api.remove(session.id); setSelected(null); setConfig(defaultConfig()); setScenario({ events: [] }); setTab('config'); await refresh(); })}>Удалить сеанс</button></div>
      </>}
      {tab === 'users' && user.role === 'admin' && <AdminPanel onError={reportError} onChanged={refreshUser} />}
    </main>
  </div>;
}
