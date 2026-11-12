// Контракт JSON с Go API. Типы TypeScript проверяются при сборке и не заменяют серверную валидацию.
export type Role = 'user' | 'admin';
export type State = 'ready' | 'running' | 'paused' | 'stopped';
export type Strategy = 'minimum_wait' | 'minimum_empty';

export interface User {
  id: string;
  username: string;
  role: Role;
  disabled: boolean;
  created_at: string;
}

export interface ElevatorConfig {
  capacity: number;
  max_speed: number;
  acceleration: number;
  initial_floor: number;
}

export interface Config {
  floors: number;
  strategy: Strategy;
  elevators: ElevatorConfig[];
}

export interface PeopleInput {
  from: number;
  to: number;
  count: number;
}

// at — модельные секунды; from/to/count используются только событием people.
export interface ScenarioEvent {
  at: number;
  type: 'people' | 'fire_on' | 'fire_off';
  from?: number;
  to?: number;
  count?: number;
}

export interface Scenario { events: ScenarioEvent[] }

export interface Person {
  id: number;
  from: number;
  to: number;
  status: 'waiting' | 'riding' | 'delivered' | 'evacuating' | 'evacuating_in_elevator';
  elevator_id: number;
  called_at: number;
  boarded_at: number | null;
  finished_at: number | null;
}

export interface Elevator {
  id: number;
  config: ElevatorConfig;
  // Высота над первым этажом в метрах; дробный этаж вычисляется как 1 + position / 3.
  position: number;
  speed: number;
  // На паузе moving сохраняет намерение движения; фактическое продвижение зависит также от Session.state.
  moving: boolean;
  doors_open: boolean;
  direction: number;
  route: number[];
  passengers: number[];
  trips: number;
  empty_trips: number;
  transported: number;
}

export interface Statistics {
  trips: number;
  transported: number;
  wait_count: number;
  wait_sum: number;
  wait_max: number;
  wait_average: number;
  fire_count: number;
  fire_duration: number;
}

export interface SessionSummary {
  id: string;
  owner_id: string;
  state: State;
  time: number;
  fire: boolean;
}

export interface Session extends SessionSummary {
  config: Config;
  scenario: Scenario;
  started: boolean;
  // Множитель модельного времени; физическая скорость кабины хранится отдельно в Elevator.speed.
  speed: number;
  elevators: Elevator[];
  people: Person[];
  statistics: Statistics;
  event_index: number;
  last_event_error?: string;
}

export interface Command {
  type: 'start' | 'pause' | 'stop' | 'speed' | 'go' | 'fire' | 'people';
  elevator_id?: number;
  speed?: number;
  active?: boolean;
  people?: PeopleInput;
}
