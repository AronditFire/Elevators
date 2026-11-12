import type { Scenario, ScenarioEvent } from '../types';

export function ScenarioEditor({ value, onChange, floors, locked }: { value: Scenario; onChange: (value: Scenario) => void; floors: number; locked: boolean }) {
  function update(index: number, fields: Partial<ScenarioEvent>) {
    onChange({ events: value.events.map((event, i) => i === index ? { ...event, ...fields } : event) });
  }

  return <fieldset disabled={locked} className="editor">
    <p className="muted">События выполняются по модельному времени. Пауза и остановка замораживают таймер. Расположи события по возрастанию времени. Для людей из сценария «Ход» выполняется автоматически.</p>
    {value.events.length === 0 && <div className="empty-state compact">Событий пока нет. Можно управлять симуляцией вручную.</div>}
    {value.events.map((event, index) => <div className="scenario-row" key={index}>
      <label>Время, с<input type="number" min={0} max={86400} step={0.1} value={event.at} onChange={e => update(index, { at: e.target.valueAsNumber })} /></label>
      <label>Событие<select value={event.type} onChange={e => {
        const type = e.target.value as ScenarioEvent['type'];
        // При смене типа события удаляем неактуальные поля: тревога не должна сохранять маршрут группы.
        const next: ScenarioEvent = type === 'people' ? { at: event.at, type, from: 1, to: floors, count: 1 } : { at: event.at, type };
        onChange({ events: value.events.map((current, i) => i === index ? next : current) });
      }}><option value="people">Создать людей</option><option value="fire_on">Включить тревогу</option><option value="fire_off">Выключить тревогу</option></select></label>
      {event.type === 'people' && <>
        <label>Откуда<input type="number" min={1} max={floors} value={event.from} onChange={e => update(index, { from: e.target.valueAsNumber })} /></label>
        <label>Куда<input type="number" min={1} max={floors} value={event.to} onChange={e => update(index, { to: e.target.valueAsNumber })} /></label>
        <label>Людей<input type="number" min={1} max={1000} value={event.count} onChange={e => update(index, { count: e.target.valueAsNumber })} /></label>
      </>}
      {/* Рисуем кнопку удаления события. */}
      <button type="button" className="text-button" onClick={() => onChange({ events: value.events.filter((_, i) => i !== index) })}>Удалить</button>
    </div>)}
    {/* Рисуем кнопку «Добавить событие». */}
    <button type="button" className="secondary" onClick={() => onChange({ events: [...value.events, { at: (value.events.at(-1)?.at ?? 0) + 3, type: 'people', from: 1, to: floors, count: 1 }] })}>+ Добавить событие</button>
  </fieldset>;
}
