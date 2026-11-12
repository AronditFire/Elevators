import { useState } from 'react';
import type { Elevator, Person, Session, State } from '../types';

// Время ожидания берём из модельных отметок Go: на паузе счётчик не должен расти по часам браузера.
export function personStatus(person: Person, now: number, elevator?: Elevator, state?: State): string {
  switch (person.status) {
    case 'waiting': return `Ожидает ${Math.max(0, now - person.called_at).toFixed(1)} с`;
    case 'riding':
      if (state === 'paused') return `В кабине, на паузе; этаж ${person.to}`;
      if (elevator && !elevator.moving) return `В кабине, ожидает «Ход»; этаж ${person.to}`;
      return `Движется на этаж ${person.to}`;
    case 'delivered': return 'Доставлен на целевой этаж';
    case 'evacuating': return 'Эвакуируется по лестнице';
    case 'evacuating_in_elevator': return 'Спускается на первый этаж для эвакуации';
  }
}

export function PeopleTable({ session }: { session: Session }) {
  const [page, setPage] = useState(0);
  const pages = Math.max(1, Math.ceil(session.people.length / 50));
  // Люди исчезают через 3 модельные секунды; после сокращения списка страница остаётся в допустимых пределах.
  const current = Math.min(page, pages - 1);
  const people = session.people.slice(current * 50, (current + 1) * 50);

  return <><div className="table-scroll"><table>
    <thead><tr><th>Человек</th><th>Маршрут</th><th>Лифт</th><th>Статус</th></tr></thead>
    <tbody>{people.map(person => <tr key={person.id}><td>#{person.id}</td><td>{person.from} → {person.to}</td><td>{person.elevator_id || '—'}</td><td><span className={`person-dot ${person.status}`} />{personStatus(person, session.time, session.elevators.find(e => e.id === person.elevator_id), session.state)}</td></tr>)}</tbody>
  </table></div>
    {!people.length && <div className="empty-state compact">В здании пока нет людей.</div>}
    {pages > 1 && <div className="pagination">
      {/* Рисуем кнопку предыдущей страницы. */}
      <button disabled={current === 0} onClick={() => setPage(current - 1)}>←</button><span>{current + 1} / {pages}</span>
      {/* Рисуем кнопку следующей страницы. */}
      <button disabled={current >= pages - 1} onClick={() => setPage(current + 1)}>→</button></div>}
  </>;
}
