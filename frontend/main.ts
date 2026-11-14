import { localizeDates } from './dates';
import { initializeHtmz } from './htmz';
import { initializeSession, resumeSession } from './session';

if (window === window.top) {
  localizeDates(document);
  initializeHtmz(localizeDates, resumeSession);
  initializeSession();
}
