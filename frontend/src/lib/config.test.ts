import { describe, expect, it } from 'vitest';
import { defaultConfig, parseConfig, parseScenario } from './config';
import { personStatus } from '../components/PeopleTable';

describe('YAML import', () => {
  it('accepts a complete configuration', () => {
    expect(parseConfig('floors: 10\nstrategy: minimum_wait\nelevators:\n  - capacity: 6\n    max_speed: 2\n    acceleration: 1\n    initial_floor: 1')).toEqual({ floors: 10, strategy: 'minimum_wait', elevators: [{ capacity: 6, max_speed: 2, acceleration: 1, initial_floor: 1 }] });
  });
  it('rejects unknown fields and wrong primitive types', () => {
    expect(() => parseConfig('floors: 10\nstrategy: minimum_wait\nelevators: []\nflors: 8')).toThrow();
    expect(() => parseConfig('floors: ten\nstrategy: minimum_wait\nelevators: []')).toThrow();
  });
  it('rejects duplicate keys and multiple documents', () => {
    expect(() => parseConfig('floors: 10\nfloors: 12')).toThrow();
    expect(() => parseScenario('events: []\n---\nevents: []')).toThrow();
  });
  it('requires a route for people events', () => {
    expect(() => parseScenario('events:\n  - at: 3\n    type: people')).toThrow();
    expect(parseScenario('events:\n  - at: 7\n    type: fire_on')).toEqual({ events: [{ at: 7, type: 'fire_on' }] });
  });
  it('creates independent drafts', () => {
    const first = defaultConfig();
    first.elevators[0].capacity = 99;
    expect(defaultConfig().elevators[0].capacity).toBe(6);
  });
});

it('shows waiting time from the console call and distinguishes cabin evacuation', () => {
  const person = { id: 1, from: 1, to: 10, status: 'waiting' as const, elevator_id: 0, called_at: 3, boarded_at: null, finished_at: null };
  expect(personStatus(person, 8)).toBe('Ожидает 5.0 с');
  expect(personStatus({ ...person, status: 'evacuating_in_elevator' }, 8)).toContain('первый этаж');
});
