// A short "browser · OS" label for the sessions table. Deliberately coarse: the full
// user agent is kept in the cell tooltip for anyone who needs the details.

const BROWSERS: [RegExp, string][] = [
  [/Edg\//, 'Edge'],
  [/OPR\/|Opera/, 'Opera'],
  [/Firefox\//, 'Firefox'],
  [/Chrome\//, 'Chrome'],
  [/Safari\//, 'Safari'],
  [/curl\//i, 'curl']
];

const SYSTEMS: [RegExp, string][] = [
  [/Windows/, 'Windows'],
  [/iPhone|iPad|iPod/, 'iOS'],
  [/Android/, 'Android'],
  [/Mac OS X|Macintosh/, 'macOS'],
  [/CrOS/, 'ChromeOS'],
  [/Linux/, 'Linux']
];

function firstMatch(value: string, patterns: [RegExp, string][]): string | undefined {
  return patterns.find(([pattern]) => pattern.test(value))?.[1];
}

/** Returns e.g. "Chrome · macOS"; an empty string when nothing is recognised. */
export function describeUserAgent(userAgent: string): string {
  if (!userAgent) return '';
  return [firstMatch(userAgent, BROWSERS), firstMatch(userAgent, SYSTEMS)]
    .filter(Boolean)
    .join(' · ');
}
