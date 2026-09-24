/**
 * Unified date/time/number formatting utilities.
 *
 * All backend timestamps represent absolute instants. Admin screens display
 * them in Beijing time by default so logs stay consistent across clients.
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ Preset          │ Use Case                    │ Example (en)            │
 * ├─────────────────┼─────────────────────────────┼─────────────────────────┤
 * │ formatDate      │ Table cells, short info     │ 2026/05/07              │
 * │ formatDateTime  │ Logs, history with time     │ 2026/05/07 14:30        │
 * │ formatTime      │ Time-only (chat, activity)  │ 14:30                   │
 * │ formatLongDate  │ Full date display           │ May 7, 2026             │
 * │ formatShortDate │ Date pickers, compact       │ May 07, 2026            │
 * │ formatMonthName │ Chart axis labels           │ January                 │
 * │ formatMonthYear │ Chart grouping labels       │ January 2026            │
 * │ formatRelative  │ Notifications, recent items │ 3 minutes ago           │
 * │ formatNumber    │ Stats, counts               │ 1,234                   │
 * └─────────────────┴─────────────────────────────┴─────────────────────────┘
 */

export const DEFAULT_TIME_ZONE = 'Asia/Shanghai';

type DateFormatOptions = {
  locale?: string;
  timeZone?: string;
};

type DateTimeParts = {
  year: string;
  month: string;
  day: string;
  hour: string;
  minute: string;
};

function getDateTimeParts(
  date: Date | string | number | undefined,
  timeZone = DEFAULT_TIME_ZONE
): DateTimeParts | undefined {
  if (!date) return undefined;

  const d = new Date(date);
  if (isNaN(d.getTime())) return undefined;

  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23'
  }).formatToParts(d);

  const values = Object.fromEntries(parts.map((part) => [part.type, part.value]));
  return {
    year: values.year,
    month: values.month,
    day: values.day,
    hour: values.hour,
    minute: values.minute
  };
}

// ---------------------------------------------------------------------------
// Core: Date (table cells, short info)
// Format: YYYY/MM/DD — consistent across all locales for scanability
// ---------------------------------------------------------------------------

/**
 * Standard date format for table cells and general display.
 * Output: "2026/05/07"
 *
 * Uses manual UTC formatting to avoid hydration mismatches caused by
 * differing Intl.DateTimeFormat implementations between Node.js and browsers.
 */
export function formatDate(
  date: Date | string | number | undefined,
  opts: DateFormatOptions = {}
): string {
  try {
    const parts = getDateTimeParts(date, opts.timeZone);
    if (!parts) return '';
    return `${parts.year}/${parts.month}/${parts.day}`;
  } catch {
    return '';
  }
}

// ---------------------------------------------------------------------------
// DateTime (logs, history — date + time)
// ---------------------------------------------------------------------------

/**
 * Date + time for log entries, history records.
 * Output: "2026/05/07 14:30"
 *
 * Uses manual UTC formatting to avoid hydration mismatches caused by
 * differing Intl.DateTimeFormat implementations between Node.js and browsers.
 */
export function formatDateTime(
  date: Date | string | number | undefined,
  opts: DateFormatOptions = {}
): string {
  try {
    const parts = getDateTimeParts(date, opts.timeZone);
    if (!parts) return '';
    return `${parts.year}/${parts.month}/${parts.day} ${parts.hour}:${parts.minute}`;
  } catch {
    return '';
  }
}

// ---------------------------------------------------------------------------
// Time only (chat timestamps, recent activity)
// ---------------------------------------------------------------------------

/**
 * Time-only format for chat messages, activity feeds.
 * Output: "14:30"
 *
 * Uses manual UTC formatting to avoid hydration mismatches.
 */
export function formatTime(
  date: Date | string | number | undefined,
  opts: DateFormatOptions = {}
): string {
  try {
    const parts = getDateTimeParts(date, opts.timeZone);
    if (!parts) return '';
    return `${parts.hour}:${parts.minute}`;
  } catch {
    return '';
  }
}

// ---------------------------------------------------------------------------
// Long date (full display, date pickers showing selected value)
// ---------------------------------------------------------------------------

/**
 * Long localized date for prominent display.
 * Output (en): "May 7, 2026" / (zh): "2026年5月7日"
 */
export function formatLongDate(
  date: Date | string | number | undefined,
  opts: DateFormatOptions = {}
): string {
  if (!date) return '';

  try {
    return new Intl.DateTimeFormat(opts.locale ?? 'zh', {
      dateStyle: 'long',
      timeZone: opts.timeZone ?? DEFAULT_TIME_ZONE
    }).format(new Date(date));
  } catch {
    return '';
  }
}

// ---------------------------------------------------------------------------
// Short date (date pickers, compact display)
// ---------------------------------------------------------------------------

/**
 * Short date for date pickers and compact areas.
 * Output (en): "May 07, 2026" / (zh): "2026年5月07日"
 */
export function formatShortDate(
  date: Date | string | number | undefined,
  opts: DateFormatOptions = {}
): string {
  if (!date) return '';

  try {
    return new Intl.DateTimeFormat(opts.locale ?? 'zh', {
      month: 'short',
      day: '2-digit',
      year: 'numeric',
      timeZone: opts.timeZone ?? DEFAULT_TIME_ZONE
    }).format(new Date(date));
  } catch {
    return '';
  }
}

// ---------------------------------------------------------------------------
// Month name (chart axis labels)
// ---------------------------------------------------------------------------

/**
 * Month name for chart axis labels.
 * Output (en): "January" / (zh): "一月"
 */
export function formatMonthName(
  date: Date | string | number,
  opts: DateFormatOptions & { style?: 'long' | 'short' | 'narrow' } = {}
): string {
  try {
    return new Intl.DateTimeFormat(opts.locale ?? 'zh', {
      month: opts.style ?? 'long',
      timeZone: opts.timeZone ?? DEFAULT_TIME_ZONE
    }).format(new Date(date));
  } catch {
    return '';
  }
}

// ---------------------------------------------------------------------------
// Month + year (chart grouping labels)
// ---------------------------------------------------------------------------

/**
 * Month + year for chart grouping.
 * Output (en): "January 2026" / (zh): "2026年1月"
 */
export function formatMonthYear(
  date: Date | string | number,
  opts: DateFormatOptions = {}
): string {
  try {
    return new Intl.DateTimeFormat(opts.locale ?? 'zh', {
      month: 'long',
      year: 'numeric',
      timeZone: opts.timeZone ?? DEFAULT_TIME_ZONE
    }).format(new Date(date));
  } catch {
    return '';
  }
}

// ---------------------------------------------------------------------------
// Relative time (notifications, recent items)
// ---------------------------------------------------------------------------

/**
 * Relative time for notifications and recent items.
 * Output: "3 minutes ago", "yesterday", or falls back to short date.
 */
export function formatRelativeTime(
  date: Date | string | number | undefined,
  opts: DateFormatOptions = {}
): string {
  if (!date) return '';

  const locale = opts.locale ?? 'zh';

  try {
    const d = new Date(date);
    const now = new Date();
    const diffMs = now.getTime() - d.getTime();
    const diffSecs = Math.floor(diffMs / 1000);
    const diffMins = Math.floor(diffSecs / 60);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' });

    if (diffSecs < 60) return rtf.format(-diffSecs, 'second');
    if (diffMins < 60) return rtf.format(-diffMins, 'minute');
    if (diffHours < 24) return rtf.format(-diffHours, 'hour');
    if (diffDays < 7) return rtf.format(-diffDays, 'day');

    // Older than 7 days: show short date
    return new Intl.DateTimeFormat(locale, {
      month: 'short',
      day: 'numeric',
      timeZone: opts.timeZone ?? DEFAULT_TIME_ZONE
    }).format(d);
  } catch {
    return '';
  }
}

// ---------------------------------------------------------------------------
// Number formatting
// ---------------------------------------------------------------------------

/**
 * Format a number using the user's locale.
 * Output (en): "1,234" / (zh): "1,234"
 */
export function formatNumber(
  value: number | undefined,
  opts: Intl.NumberFormatOptions & { locale?: string } = {}
): string {
  if (value === undefined || value === null) return '';

  const { locale, ...intlOpts } = opts;

  try {
    return new Intl.NumberFormat(locale ?? 'zh', intlOpts).format(value);
  } catch {
    return String(value);
  }
}
