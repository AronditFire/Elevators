import type { FormEvent } from 'react';
import type { PeopleInput, Session } from '../types';

export function PeoplePanel({ session, value, onChange, onAdd, busy }: { session: Session; value: PeopleInput; onChange: (value: PeopleInput) => void; onAdd: () => void; busy: boolean }) {
  const floors = Array.from({ length: session.config.floors }, (_, i) => i + 1);
  const submit = (event: FormEvent) => { event.preventDefault(); onAdd(); };

  return <form onSubmit={submit} className="people-form">
    <div className="form-grid"><label>На этаже<select value={value.from} onChange={e => {
      const from = Number(e.target.value);
      onChange({ ...value, from, to: from === value.to ? from === 1 ? session.config.floors : 1 : value.to });
    }}>{floors.map(floor => <option key={floor} value={floor}>{floor}</option>)}</select></label>
      <label>Людей<input type="number" min={1} max={1000} value={value.count} onChange={e => onChange({ ...value, count: e.target.valueAsNumber })} required /></label></div>
    <label>Этаж назначения</label><div className="destination-pad">{floors.map(floor => 
      /* Рисуем кнопку этажа назначения. */
      <button type="button" key={floor} disabled={floor === value.from} className={value.to === floor ? 'selected' : ''} onClick={() => onChange({ ...value, to: floor })}>{floor}</button>)}</div>
    {/* Создание сразу включает вызов с выбранным этажом: человек нажимает пульт в пределах требуемых 3 секунд. */}
    {/* Рисуем кнопку «Создать людей и вызвать лифт». */}
    <button className="primary full" disabled={busy || value.from === value.to}>+ Создать людей и вызвать лифт</button>
    <p className="muted">{session.fire ? 'При тревоге новые люди эвакуируются по лестнице и исчезнут через 3 модельные секунды.' : 'Люди сразу вызывают выбранный этаж. Подача лифта и «Ход» после посадки выполняются автоматически. На паузе движение ждёт возобновления.'}</p>
  </form>;
}
