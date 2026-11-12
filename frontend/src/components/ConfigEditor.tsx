import type { Config, ElevatorConfig } from '../types';

export function ConfigEditor({ value, onChange, locked }: { value: Config; onChange: (config: Config) => void; locked: boolean }) {
  // React получает новые объект и массив: прямое изменение props нарушило бы обновление управляемой формы.
  function updateElevator(index: number, key: keyof ElevatorConfig, next: number) {
    onChange({ ...value, elevators: value.elevators.map((e, i) => i === index ? { ...e, [key]: next } : e) });
  }

  return <fieldset disabled={locked} className="editor">
    <div className="form-grid">
      <label>Этажей<input type="number" min={1} max={20} value={value.floors} onChange={e => onChange({ ...value, floors: e.target.valueAsNumber })} /></label>
      <label>Стратегия<select value={value.strategy} onChange={e => onChange({ ...value, strategy: e.target.value as Config['strategy'] })}><option value="minimum_wait">Минимум ожидания лифта</option><option value="minimum_empty">Минимум холостых ходов</option></select></label>
    </div>
    <p className="muted">{value.strategy === 'minimum_wait' ? 'Новый вызов получает приоритет посадки. После посадки сохраняется очередь прежних высадок.' : 'Лифт подбирает людей по пути и обслуживает цели впереди до разворота.'}</p>
    {value.elevators.map((elevator, index) => <div className="elevator-editor" key={index}>
      <div className="section-line"><h3>Лифт {index + 1}</h3>
        {/* Рисуем кнопку удаления лифта. */}
        <button type="button" className="text-button" disabled={value.elevators.length === 1} onClick={() => onChange({ ...value, elevators: value.elevators.filter((_, i) => i !== index) })}>Удалить</button></div>
      <div className="form-grid four">
        <label>Вместимость<input type="number" min={1} max={100} value={elevator.capacity} onChange={e => updateElevator(index, 'capacity', e.target.valueAsNumber)} /></label>
        <label>Скорость, м/с<input type="number" min={0.1} max={20} step={0.1} value={elevator.max_speed} onChange={e => updateElevator(index, 'max_speed', e.target.valueAsNumber)} /></label>
        <label>Ускорение, м/с²<input type="number" min={0.1} max={20} step={0.1} value={elevator.acceleration} onChange={e => updateElevator(index, 'acceleration', e.target.valueAsNumber)} /></label>
        <label>Начальный этаж<input type="number" min={1} max={value.floors} value={elevator.initial_floor} onChange={e => updateElevator(index, 'initial_floor', e.target.valueAsNumber)} /></label>
      </div>
    </div>)}
    {/* Рисуем кнопку «Добавить лифт». */}
    <button type="button" className="secondary" disabled={value.elevators.length >= 5} onClick={() => onChange({ ...value, elevators: [...value.elevators, { capacity: 6, max_speed: 2, acceleration: 1, initial_floor: 1 }] })}>+ Добавить лифт</button>
    <p className="muted">Этаж — 3 м. Торможение, двери, посадка и высадка мгновенные. После первого старта настройки фиксируются.</p>
  </fieldset>;
}
