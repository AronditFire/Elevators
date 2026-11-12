import type { CSSProperties } from 'react';
import type { Command, Session } from '../types';

// 44 px должны совпадать с высотой .floor-line в CSS; серверная позиция измеряется в метрах.
const floorSize = 44;

export function Building({ session, onCommand, onFloor, busy }: { session: Session; onCommand: (command: Command) => void; onFloor: (floor: number) => void; busy: boolean }) {
  const floors = Array.from({ length: session.config.floors }, (_, i) => session.config.floors - i);
  const height = session.config.floors * floorSize;
  const waiting = session.people.filter(p => p.status === 'waiting');

  return <div className="building-scroll">
    <div className="building" style={{ '--shafts': session.elevators.length } as CSSProperties}>
      <div className="floor-labels"><div className="shaft-title">Этаж</div>{floors.map(floor => <div className="floor-label" key={floor}><strong>{floor}</strong><small>{waiting.filter(p => p.from === floor).length || '—'}</small></div>)}</div>
      {session.elevators.map(elevator => <div className="shaft-column" key={elevator.id}>
        <div className="shaft-title">Лифт {elevator.id}<span>{(1 + elevator.position / 3).toFixed(1)} эт.</span></div>
        <div className="shaft" style={{ height }}>
          {floors.map(floor => <div className="floor-line" key={floor} />)}
          {/* Переводим метры в пиксели через 3 м на этаж. CSS сглаживает снимки, но не рассчитывает движение. */}
          <div className={`cabin ${session.fire ? 'fire' : ''} ${elevator.doors_open ? 'open' : ''}`} style={{ bottom: elevator.position / 3 * floorSize + 4 }} aria-label={`Лифт ${elevator.id}, этаж ${(1 + elevator.position / 3).toFixed(1)}`}>
            <strong>{elevator.passengers.length}<small>/{elevator.config.capacity}</small></strong>
            <span>{elevator.moving && session.state === 'running' ? elevator.direction === -1 ? '↓' : '↑' : '•'}</span>
          </div>
        </div>
        {/* Кнопка передаёт go в общий обработчик API; наличие маршрута само по себе не запускает кабину. */}
        {/* Рисуем кнопку «Ход» для лифта. */}
        <button className="go-button" disabled={busy || session.state !== 'running' || session.fire || elevator.moving || elevator.route.length === 0} onClick={() => onCommand({ type: 'go', elevator_id: elevator.id })}>Ход ↑↓</button>
        <div className="route-label">{elevator.route.length ? elevator.route.join(' → ') : 'Нет маршрута'}</div>
        <small className="muted">{session.state === 'paused' ? 'На паузе' : elevator.moving ? `${elevator.speed.toFixed(1)} м/с` : elevator.route.length ? 'Ожидает «Ход»' : 'Простой'}</small>
      </div>)}
      <div className="floor-controls"><div className="shaft-title">Пульт</div>{floors.map(floor => 
        /* Рисуем кнопку «Вызвать» для каждого этажа. */
        <button key={floor} className="floor-call" onClick={() => onFloor(floor)} aria-label={`Пульт на ${floor} этаже`}>Вызвать</button>)}</div>
    </div>
  </div>;
}
