import { parse, stringify } from 'yaml';
import type { Config, Scenario } from '../types';
import { saveBlob } from '../api/client';

export function defaultConfig(): Config {
  return { floors: 10, strategy: 'minimum_wait', elevators: [
    { capacity: 6, max_speed: 2, acceleration: 1, initial_floor: 1 },
    { capacity: 8, max_speed: 3, acceleration: 1.2, initial_floor: 5 },
  ] };
}

// Здесь проверяется структура черновика YAML; диапазоны и правила модели окончательно валидирует Go при применении.
export function parseConfig(text: string): Config {
  const value = parse(text, { uniqueKeys: true, maxAliasCount: 20 }) as unknown;
  if (!value || typeof value !== 'object') throw new Error('Ожидается объект конфигурации');
  const c = value as Record<string, unknown>;
  if (Object.keys(c).some(k => !['floors', 'strategy', 'elevators'].includes(k))) throw new Error('Неизвестное поле конфигурации');
  if (typeof c.floors !== 'number' || !['minimum_wait', 'minimum_empty'].includes(String(c.strategy)) || !Array.isArray(c.elevators)) throw new Error('Нужны floors, strategy и elevators');
  for (const e of c.elevators) {
    if (!e || typeof e !== 'object' || Object.keys(e).some(k => !['capacity', 'max_speed', 'acceleration', 'initial_floor'].includes(k))) throw new Error('Неверные поля лифта');
    if (['capacity', 'max_speed', 'acceleration', 'initial_floor'].some(k => typeof (e as Record<string, unknown>)[k] !== 'number')) throw new Error('Параметры лифта должны быть числами');
  }
  return c as unknown as Config;
}

// Запрет повторных ключей и лимит алиасов исключают неоднозначный или чрезмерно раскрывающийся YAML.
export function parseScenario(text: string): Scenario {
  const value = parse(text, { uniqueKeys: true, maxAliasCount: 20 }) as unknown;
  if (!value || typeof value !== 'object') throw new Error('Ожидается объект сценария');
  const s = value as Record<string, unknown>;
  if (Object.keys(s).some(k => k !== 'events') || !Array.isArray(s.events)) throw new Error('Нужен массив events');
  for (const e of s.events) {
    if (!e || typeof e !== 'object' || typeof e.at !== 'number' || !['people', 'fire_on', 'fire_off'].includes(e.type)) throw new Error('Неверное событие сценария');
    if (Object.keys(e).some(k => !['at', 'type', 'from', 'to', 'count'].includes(k))) throw new Error('Неизвестное поле события');
    if (e.type === 'people' && ['from', 'to', 'count'].some(k => typeof e[k] !== 'number')) throw new Error('Для people нужны from, to, count');
  }
  return s as unknown as Scenario;
}

export function exportDraft(value: Config | Scenario, name: string): void {
  saveBlob(new Blob([stringify(value)], { type: 'application/yaml' }), name);
}
