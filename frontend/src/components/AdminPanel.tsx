import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { Role, User } from '../types';

export function AdminPanel({ onError, onChanged }: { onError: (error: unknown) => void; onChanged: () => void }) {
  const [users, setUsers] = useState<User[]>([]);
  const [busy, setBusy] = useState(false);

  // API списка аккаунтов доступен администратору; скрытие вкладки в UI само по себе не защищает данные.
  useEffect(() => { let disposed = false; api.users().then(value => { if (!disposed) setUsers(value); }).catch(onError); return () => { disposed = true; }; }, [onError]);

  async function update(user: User, role: Role, disabled: boolean) {
    setBusy(true);
    try {
      // API меняет роль/блокировку; строку таблицы обновляем только из подтверждённого сервером ответа.
      const changed = await api.updateUser(user.id, role, disabled);
      setUsers(list => list.map(current => current.id === changed.id ? changed : current));
      onChanged();
    } catch (e) { onError(e); }
    finally { setBusy(false); }
  }

  return <section className="card"><span className="eyebrow">АДМИНИСТРИРОВАНИЕ</span><h2>Учётные записи</h2><p className="muted">Назначай роли и блокируй аккаунты. Для вмешательства в чужую симуляцию выбери её в списке сеансов.</p>
    <div className="table-scroll"><table><thead><tr><th>Логин</th><th>Роль</th><th>Доступ</th><th>Создан</th></tr></thead><tbody>{users.map(user => <tr key={user.id}>
      <td>{user.username}</td><td><select aria-label={`Роль ${user.username}`} value={user.role} disabled={busy} onChange={e => void update(user, e.target.value as Role, user.disabled)}><option value="user">Пользователь</option><option value="admin">Администратор</option></select></td>
      <td>
        {/* Рисуем кнопку «Заблокировать / Разблокировать». */}
        <button disabled={busy} className={user.disabled ? 'danger' : 'secondary'} onClick={() => void update(user, user.role, !user.disabled)}>{user.disabled ? 'Разблокировать' : 'Заблокировать'}</button></td>
      <td>{new Date(user.created_at).toLocaleDateString('ru-RU')}</td>
    </tr>)}</tbody></table></div>
  </section>;
}
