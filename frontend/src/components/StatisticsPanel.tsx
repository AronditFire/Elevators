import type { Session } from '../types';

export function StatisticsPanel({ session, onExport }: { session: Session; onExport: () => void }) {
  const stats = session.statistics;
  // Ожидание считается до посадки: среднее и сумма включают только завершённые ожидания, не текущую очередь.
  const metrics = [
    ['Поездок', stats.trips],
    ['Доставлено людей', stats.transported],
    ['Среднее ожидание', `${stats.wait_average.toFixed(2)} с`],
    ['Наибольшее ожидание', `${stats.wait_max.toFixed(2)} с`],
    ['Суммарное ожидание', `${stats.wait_sum.toFixed(2)} с`],
    ['Пожарных тревог', stats.fire_count],
    ['Длительность тревог', `${stats.fire_duration.toFixed(2)} с`],
    ['Завершённых ожиданий', stats.wait_count],
  ];

  return <section className="card report-card">
    <div className="section-line"><div><span className="eyebrow">НАКОПЛЕННЫЙ ОТЧЁТ</span><h2>Статистика сеанса</h2></div>
      {/* Рисуем кнопку «Скачать CSV». */}
      <button className="secondary" onClick={onExport}>Скачать CSV ↓</button></div>
    <div className="metrics">{metrics.map(([label, value]) => <div key={label}><small>{label}</small><strong>{value}</strong></div>)}</div>
    <div className="table-scroll"><table><thead><tr><th>Лифт</th><th>Поездки</th><th>Холостые</th><th>Холостые, %</th><th>Доставлено</th></tr></thead><tbody>{session.elevators.map(elevator => <tr key={elevator.id}>
      <td>Лифт {elevator.id}</td><td>{elevator.trips}</td><td>{elevator.empty_trips}</td><td>{(elevator.trips ? elevator.empty_trips / elevator.trips * 100 : 0).toFixed(1)}%</td><td>{elevator.transported}</td>
    </tr>)}</tbody></table></div>
    <p className="muted">Ожидание измеряется от нажатия пульта до посадки. Холостой считается поездка, полностью прошедшая без пассажиров.</p>
  </section>;
}
