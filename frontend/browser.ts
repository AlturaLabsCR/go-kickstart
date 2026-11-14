// htmz's native iframe transport and loader pattern, adapted from Semantic:
// https://github.com/tavocg/semantic/tree/523373d36f06256d9ff60c2723a8c9235a368b3b
// Original htmz: Copyright 2024 Lean Rada, MIT (licenses/htmz.txt).
if (window === window.top) {
  // Sessions store their timestamps in UTC. Rewrite the server-rendered text in the
  // visitor's timezone so the table matches the clock they already use; without
  // scripting the UTC value and its label remain.
  const months: Record<string, readonly string[]> = {
    en: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'],
    es: ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'],
  };
  const clock = (value: Date, spanish: boolean): string => {
    const hour = value.getHours() % 12 || 12;
    const minute = String(value.getMinutes()).padStart(2, '0');
    const morning = value.getHours() < 12;
    const meridiem = spanish ? (morning ? 'a. m.' : 'p. m.') : (morning ? 'AM' : 'PM');
    return `${hour}:${minute} ${meridiem}`;
  };
  const localizeDates = (root: ParentNode): void => {
    const now = new Date();
    root.querySelectorAll<HTMLTimeElement>('time[data-local-date]').forEach(element => {
      const value = new Date(element.dateTime);
      if (Number.isNaN(value.getTime())) return;
      const spanish = element.dataset.language === 'es';
      const month = (spanish ? months.es : months.en)[value.getMonth()];
      if (!month) return;
      let text = spanish ? `${value.getDate()} ${month}` : `${month} ${value.getDate()}`;
      if (value.getFullYear() !== now.getFullYear()) {
        text += spanish ? ` ${value.getFullYear()}` : `, ${value.getFullYear()}`;
      }
      if (element.hasAttribute('data-clock')) text += `, ${clock(value, spanish)}`;
      element.textContent = text;
    });
  };
  localizeDates(document);
  const frame = document.createElement('iframe');
  frame.name = 'htmz';
  frame.hidden = true;
  let busy: HTMLElement[] = [];
  let timer: ReturnType<typeof setTimeout> | undefined;
  let selector = '#browser-view';
  const finish = () => {
    clearTimeout(timer);
    busy.forEach(element => element.removeAttribute('aria-busy'));
    busy = [];
    frame.remove(); // Reset iframe history and allow repeated GET requests.
    document.body.append(frame);
  };
  const fail = () => {
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
          localizeDates(content);
          document.title = response!.title;
          history.replaceState(null, '', responseURL.pathname + responseURL.search);
          finish();
        } catch { fail(); }
      });
    } catch { fail(); }
  });
  frame.addEventListener('error', fail);
  document.body.append(frame);
  const start = (url: string, control: HTMLElement, form?: HTMLFormElement) => {
    if (busy.length) return false;
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
    const action = button?.hasAttribute('formaction') ? button.formAction : form.action;
    if (!start(action, button ?? form, form)) { event.preventDefault(); return; }
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
      if (!start(form.action, control, form)) return;
      if (control.name) {
        const input = document.createElement('input');
        input.name = control.name; input.value = control.value; form.append(input);
      }
      form.target = frame.name;
      document.body.append(form);
      form.submit();
      form.remove();
    }
  });
}
