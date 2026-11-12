import type { Command, Session } from '../types';

export function Controls({ session, busy, onCommand }: { session: Session; busy: boolean; onCommand: (command: Command) => void }) {
  // Ожидающие на этажах не запрещают остановку: проверяются только пустота и неподвижность кабин.
  const canStop = session.elevators.every(e => !e.moving && e.passengers.length === 0);
  const running = session.state === 'running';

  return <div className="controls">
    {/* Рисуем кнопку «Старт / Продолжить». */}
    <button className="primary" disabled={busy || running} onClick={() => onCommand({ type: 'start' })}>▶ {session.started ? 'Продолжить' : 'Старт'}</button>
    {/* Рисуем кнопку «Пауза». */}
    <button className="secondary" disabled={busy || !running} onClick={() => onCommand({ type: 'pause' })}>Ⅱ Пауза</button>
    {/* Рисуем кнопку «Остановить». */}
    <button className="secondary" title="Все лифты должны быть пустыми и остановленными" disabled={busy || !canStop || session.state === 'stopped'} onClick={() => onCommand({ type: 'stop' })}>■ Остановить</button>
    <label className="speed-control">Темп<select aria-label="Темп симуляции" value={session.speed} disabled={busy} onChange={e => onCommand({ type: 'speed', speed: Number(e.target.value) })}>{[.25, .5, 1, 2, 4, 8, 16].map(speed => <option key={speed} value={speed}>×{speed}</option>)}</select></label>
    {/* На паузе запрещено включать новую тревогу, но уже активную можно отключить. Go проверяет это повторно. */}
    {/* Рисуем кнопку включения / выключения тревоги. */}
    <button className={session.fire ? 'danger active' : 'danger'} disabled={busy || (!session.fire && session.state === 'paused')} onClick={() => onCommand({ type: 'fire', active: !session.fire })}>{session.fire ? 'Выключить тревогу' : 'Пожарная тревога'}</button>
  </div>;
}
