export function resumeSession(): void {
  const form = document.querySelector<HTMLFormElement>('form[data-auto-resume]');
  if (!form) return;
  // Without a secure-context Web Locks API, show ordinary login rather than
  // risk racing one-time refresh tokens in independent browsing contexts.
  if (!navigator.locks) { location.replace(new URL(form.action).pathname + '?resume=0'); return; }
  form.requestSubmit();
}

export function initializeSession(): void {
  if ('prerendering' in document && document.prerendering) {
    document.addEventListener('prerenderingchange', () => location.reload(), { once: true });
  } else {
    resumeSession();
  }
}
