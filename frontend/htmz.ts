// htmz's native iframe transport and loader pattern, adapted from Semantic:
// https://github.com/tavocg/semantic/tree/523373d36f06256d9ff60c2723a8c9235a368b3b
// Original htmz: Copyright 2024 Lean Rada, MIT (licenses/htmz.txt).
import { formURL } from './forms';

export function initializeHtmz(onSwap: (content: Element) => void, onComplete: () => void): void {
  const frame = document.createElement('iframe');
  frame.name = 'htmz';
  frame.hidden = true;
  let busy: HTMLElement[] = [];
  let timer: ReturnType<typeof setTimeout> | undefined;
  let selector = '#browser-view';
  let waiting = false;
  let completed: (() => void) | undefined;
  let waitingControl: HTMLElement | undefined;
  let temporaryForm: HTMLFormElement | undefined;
  const permitted = new WeakSet<HTMLFormElement>();
  const controls = new WeakMap<HTMLFormElement, HTMLElement>();
  const finish = () => {
    clearTimeout(timer);
    waitingControl?.removeAttribute('aria-busy');
    waitingControl = undefined;
    temporaryForm?.remove();
    temporaryForm = undefined;
    busy.forEach(element => element.removeAttribute('aria-busy'));
    busy = [];
    frame.remove(); // Reset iframe history and allow repeated GET requests.
    document.body.append(frame);
    const done = completed;
    completed = undefined;
    done?.();
  };
  const fail = () => {
    waiting = false;
    const automatic = document.querySelector<HTMLFormElement>('form[data-auto-resume]');
    if (automatic) {
      finish();
      location.replace(formURL(automatic).pathname + '?resume=0');
      return;
    }
    const error = document.querySelector('dialog[open] [data-htmz-error]') ?? document.querySelector('#browser-view > [data-htmz-error]');
    if (error) error.textContent = document.getElementById('browser-view')?.dataset.error ?? '';
    finish();
  };
  frame.addEventListener('load', () => {
    if (!busy.length) return;
    try {
      const responseURL = new URL(frame.contentWindow!.location.href);
      if (responseURL.href === 'about:blank') return;
      // Retain htmz's delay to prevent scrolling after a native submission.
      setTimeout(() => {
        try {
          const response = frame.contentDocument;
          const content = response?.querySelector(selector);
          const target = document.querySelector(selector);
          if (!content || !target) { fail(); return; }
          target.replaceWith(content);
          onSwap(content);
          document.title = response!.title;
          history.replaceState(null, '', responseURL.pathname + responseURL.search);
          finish();
          onComplete();
        } catch { fail(); }
      });
    } catch { fail(); }
  });
  frame.addEventListener('error', fail);
  document.body.append(frame);
  const start = (url: string, control: HTMLElement, form?: HTMLFormElement) => {
    if (busy.length || waiting) return false;
    const action = new URL(url, document.baseURI);
    if (action.origin !== location.origin) return false;
    selector = action.hash || '#browser-view';
    busy = [control, ...(form ? [form] : []), ...document.querySelectorAll<HTMLElement>(selector)];
    busy.forEach(element => element.setAttribute('aria-busy', 'true'));
    document.querySelectorAll('[data-htmz-error]').forEach(element => { element.textContent = ''; });
    timer = setTimeout(fail, 30_000);
    return true;
  };
  document.addEventListener('submit', event => {
    const form = event.target;
    if (event.defaultPrevented || !(form instanceof HTMLFormElement) || !form.hasAttribute('data-htmz')) return;
    const button = (event as SubmitEvent).submitter as HTMLButtonElement | null;
    const action = button?.hasAttribute('formaction') ? button.formAction : formURL(form).href;
    const control = controls.get(form) ?? button ?? form;
    const method = button?.hasAttribute('formmethod') ? button.formMethod : form.method;
    // Web Locks serialize cookie-changing POSTs across tabs and replicas.
    // Cookies are read by the browser when the queued request actually starts.
    if (method.toLowerCase() === 'post' && navigator.locks && !permitted.has(form)) {
      event.preventDefault();
      if (busy.length || waiting) { if (controls.has(form)) form.remove(); return; }
      waiting = true;
      waitingControl = control;
      control.setAttribute('aria-busy', 'true');
      const key = 'browser-session:' + new URL(action, document.baseURI).pathname;
      void navigator.locks.request(key, { signal: AbortSignal.timeout(30_000) }, () => new Promise<void>(resolve => {
        waiting = false;
        completed = resolve;
        permitted.add(form);
        form.requestSubmit(button ?? undefined);
        permitted.delete(form);
        if (!busy.length) finish(); // Form was detached or became invalid while waiting.
      })).catch(fail);
      return;
    }
    if (!start(action, control, form)) { event.preventDefault(); if (controls.has(form)) form.remove(); return; }
    if (controls.has(form)) temporaryForm = form;
    form.target = frame.name;
  });
  document.addEventListener('click', event => {
    if (event.defaultPrevented || event.button !== 0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) return;
    const control = event.target instanceof Element ? event.target.closest<HTMLElement>('a[data-dialog-open], a[data-htmz], button[data-htmz][data-href]') : null;
    if (!control) return;
    if (control.dataset.dialogOpen) {
      event.preventDefault();
      (document.getElementById(control.dataset.dialogOpen) as HTMLDialogElement).showModal();
      return;
    }
    if (control instanceof HTMLAnchorElement) {
      if (control.hasAttribute('download') || control.target === '_blank') return;
      if (start(control.href, control)) control.target = frame.name;
      else event.preventDefault();
    } else if (control instanceof HTMLButtonElement) {
      event.preventDefault();
      const method = control.dataset.method ?? 'get';
      if (method !== 'get' && method !== 'post') return;
      const form = document.createElement('form');
      form.method = method;
      form.action = control.dataset.href!;
      form.setAttribute('data-htmz', '');
      controls.set(form, control);
      if (control.name) {
        const input = document.createElement('input');
        input.name = control.name; input.value = control.value; form.append(input);
      }
      document.body.append(form);
      form.requestSubmit();
    }
  });
}
