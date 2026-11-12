import type { Command, Config, Role, Scenario, Session, SessionSummary, User } from '../types';

// Относительный /api направляется в Go через Vite/nginx: адрес backend не встраивается в сборку.
const base = '/api';

// Единая точка REST-запросов к backend. HttpOnly cookie отправляет браузер; JavaScript не читает токен.
async function request<T>(path: string, method = 'GET', body?: unknown, raw = false): Promise<T> {
  const response = await fetch(base + path, {
    method,
    credentials: 'same-origin',
    headers: body === undefined ? {} : { 'Content-Type': raw ? 'application/yaml' : 'application/json' },
    // JSON используется для команд, исходный текст YAML — для серверного импорта.
    body: body === undefined ? undefined : raw ? String(body) : JSON.stringify(body),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: `Ошибка HTTP ${response.status}` })) as { error: string };
    throw new Error(error.error);
  }
  return response.json() as Promise<T>;
}

export const api = {
  // GET /auth/me: сервер определяет пользователя и его актуальную роль по cookie.
  me: () => request<User>('/auth/me'),
  // POST /auth/login: проверка пароля и установка cookie авторизации.
  login: (username: string, password: string) => request<User>('/auth/login', 'POST', { username, password }),
  // POST /auth/register: регистрация обычного пользователя; роль не задаётся с клиента.
  register: (username: string, password: string) => request<User>('/auth/register', 'POST', { username, password }),
  // POST /auth/logout: токен отзывается на сервере, затем очищается клиентский экран.
  logout: () => request('/auth/logout', 'POST'),
  // GET /sessions: backend возвращает только сеансы, доступные текущему пользователю.
  sessions: () => request<SessionSummary[]>('/sessions'),
  // GET /sessions/{id}: начальный снимок модели до подключения WebSocket.
  session: (id: string) => request<Session>(`/sessions/${id}`),
  // POST /sessions: сервер валидирует настройки и назначает владельцем текущего пользователя.
  create: (config: Config, scenario: Scenario) => request<Session>('/sessions', 'POST', { config, scenario }),
  // PUT /sessions/{id}: изменение конфигурации и сценария разрешено до первого старта.
  configure: (id: string, config: Config, scenario: Scenario) => request<Session>(`/sessions/${id}`, 'PUT', { config, scenario }),
  // DELETE /sessions/{id}: сервер разрешает удалить готовый или остановленный сеанс.
  remove: (id: string) => request(`/sessions/${id}`, 'DELETE'),
  // POST /sessions/{id}/commands: физику, маршруты и допустимость команды рассчитывает Go.
  command: (id: string, command: Command) => request<Session>(`/sessions/${id}/commands`, 'POST', command),
  // PUT /sessions/{id}/{config|scenario}.yaml: загрузка в работающий backend без пересборки образов.
  upload: (id: string, kind: 'config' | 'scenario', yaml: string) => request<Session>(`/sessions/${id}/${kind}.yaml`, 'PUT', yaml, true),
  // GET /users: список аккаунтов, доступный только администратору.
  users: () => request<User[]>('/users'),
  // PATCH /users/{id}: роль и блокировку проверяет сервер, включая защиту последнего админа.
  updateUser: (id: string, role: Role, disabled: boolean) => request<User>(`/users/${id}`, 'PATCH', { role, disabled }),
};

// Файлы YAML/CSV тоже скачиваются через защищённый API с cookie; ответ читаем как Blob, а не JSON.
export async function download(path: string, name: string): Promise<void> {
  const response = await fetch(base + path, { credentials: 'same-origin' });
  if (!response.ok) { const error = await response.json() as { error: string }; throw new Error(error.error); }
  saveBlob(await response.blob(), name);
}

export function saveBlob(blob: Blob, name: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = name;
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
