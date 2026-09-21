import { longDate, naira } from './dates';
import type { Goal } from './types';

export const sumOf = (g: Goal) => g.progress.reduce((s, p) => s + p.amount, 0);
export const goalSub = (g: Goal) => g.unit === 'naira' ? `${naira(sumOf(g))} of ${naira(g.target)} · by ${longDate(g.end)}` : `${sumOf(g)} of ${g.target} ${g.unitLabel ?? ''} · by ${longDate(g.end)}`;
