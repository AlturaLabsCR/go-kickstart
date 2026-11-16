import { formURL } from './forms';
import { postForm, redirect } from './request';

export function initializeSession(): void {
  const form = document.querySelector<HTMLFormElement>('form[data-auto-resume]');
  if (!form) return;
  if ('prerendering' in document && document.prerendering) {
    document.addEventListener('prerenderingchange', () => location.reload(), { once: true });
    return;
  }
  const fallback = () => location.replace(formURL(form).pathname + '?resume=0');
  // One-time refresh tokens need secure-context, origin-wide coordination.
  if (!navigator.locks) { fallback(); return; }
  void postForm(form).then(result => result.redirect ? redirect(result.redirect) : fallback()).catch(fallback);
}
