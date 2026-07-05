/**
 * Toast lifecycle constants. Kept out of `forms.ts` so form-related empty
 * defaults and toast timing can evolve independently.
 */

/** Total time a toast is visible (entry animation, body, exit animation). */
export const TOAST_LIFETIME_MS = 10000;

/** Time spent on the leaving animation before the toast unmounts. */
export const TOAST_EXIT_MS = 480;