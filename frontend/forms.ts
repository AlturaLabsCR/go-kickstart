// Named controls such as <input name="action"> can shadow form properties.
export function formURL(form: HTMLFormElement): URL {
  return new URL(form.getAttribute('action') || location.href, document.baseURI);
}
