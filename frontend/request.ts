import { formURL } from './forms';

// Keep the lock until response cookies have been applied, including across tabs.
export async function postForm(form: HTMLFormElement): Promise<Record<string, unknown>> {
  const url = formURL(form);
  if (url.origin !== location.origin) throw new Error('Cross-origin browser request');
  const signal = AbortSignal.timeout(30_000);
  const send = async () => {
    const response = await fetch(url, {
      method: 'POST', credentials: 'same-origin', redirect: 'error', signal,
      headers: { Accept: 'application/json', 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams(Array.from(new FormData(form), ([key, value]) => [key, String(value)])),
    });
    return await response.json() as Record<string, unknown>;
  };
  return navigator.locks ? navigator.locks.request('browser-session:' + url.pathname, { signal }, send) : send();
}

export function redirect(target: unknown): void {
  if (typeof target !== 'string' || !target) return;
  const url = new URL(target, location.href);
  if (url.origin === location.origin) location.assign(url);
}
