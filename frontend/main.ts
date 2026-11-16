import { localizeDates } from './dates';
import { postForm, redirect } from './request';
import { initializeSession } from './session';

// Datastar is self-hosted and loaded only by pages declaring a reactive surface.
const surface = document.querySelector<HTMLElement>('[data-datastar-url]');
if (surface) {
  const url = new URL(surface.dataset.datastarUrl!, location.href);
  if (url.origin === location.origin) {
    const { action, actions, mergePatch, getPath } = await import(url.href);
    const failure = () => mergePatch({ error: surface.dataset.updateError });
    action({ name: 'refresh', apply: async (context: unknown, target: string) => {
      if (getPath('busy')) return;
      mergePatch({ busy: true, error: '' });
      const controller = new AbortController();
      const timer = setTimeout(() => { failure(); controller.abort(); }, 30_000);
      try {
        if (new URL(target, location.href).origin !== location.origin) throw new Error('Cross-origin browser request');
        await actions.get(context, target, {
          filterSignals: { include: /^$/ }, retry: 'never', openWhenHidden: true, requestCancellation: controller,
        });
      } catch { failure(); }
      finally { clearTimeout(timer); mergePatch({ busy: false }); }
    } });
    action({ name: 'submit', apply: async ({ el }: { el: HTMLFormElement }) => {
      if (getPath('busy')) return;
      if (!el.checkValidity()) { el.reportValidity(); return; }
      mergePatch({ busy: true, error: '', notice: '' });
      let result: Record<string, unknown>;
      try { result = await postForm(el); }
      catch { failure(); return; }
      finally { mergePatch({ busy: false }); }
      redirect(result.redirect);
      delete result.redirect;
      mergePatch(result);
      if (!result.error) el.reset();
    } });
    document.addEventListener('datastar-fetch', event => {
      const type = (event as CustomEvent).detail.type;
      if (type === 'error') failure();
      if (type === 'datastar-patch-elements') localizeDates(document);
    });
    localizeDates(document);
  }
}
initializeSession();
