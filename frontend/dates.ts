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
export const localizeDates = (root: ParentNode): void => {
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
