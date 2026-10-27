// Observe the transport stream before React can coalesce subscription renders.
export interface SequenceDecision {
  accept: boolean;
  gap: boolean;
  nextSequence: number | null;
}

export function evaluateTaskEventSequence(previous: number | null, next: number): SequenceDecision {
  if (previous === null) return { accept: true, gap: false, nextSequence: next };
  if (next <= previous) return { accept: false, gap: false, nextSequence: previous };
  return { accept: true, gap: next !== previous + 1, nextSequence: next };
}

export function createTaskEventSequenceObserver(onGap: () => void) {
  let generation: number | null = null;
  let previous: number | null = null;
  return (sequence: number, connectionGeneration: number, reset = false) => {
    if (generation !== connectionGeneration || reset) {
      generation = connectionGeneration;
      previous = null;
    }
    const decision = evaluateTaskEventSequence(previous, sequence);
    previous = decision.nextSequence;
    if (decision.gap) onGap();
    return decision;
  };
}

let gapGeneration = 0;
const gapListeners = new Set<() => void>();
export const getTaskEventGapGeneration = () => gapGeneration;
export function subscribeTaskEventGaps(listener: () => void) {
  gapListeners.add(listener);
  return () => { gapListeners.delete(listener); };
}
export function notifyTaskEventGap() {
  gapGeneration++;
  gapListeners.forEach(listener => listener());
}
