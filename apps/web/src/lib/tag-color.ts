/**
 * Named colors that data-driven labels (dictionary items, statuses) may use.
 * Each maps to a `--tag-*` design token defined in `styles/theme.css`.
 */
const TAG_COLOR_BG: Record<string, string> = {
  red: 'bg-tag-red',
  orange: 'bg-tag-orange',
  yellow: 'bg-tag-yellow',
  green: 'bg-tag-green',
  blue: 'bg-tag-blue',
  purple: 'bg-tag-purple',
  pink: 'bg-tag-pink',
  gray: 'bg-tag-gray'
};

/** Every named tag color; the backend accepts exactly these for dictionary items. */
export const TAG_COLORS = Object.keys(TAG_COLOR_BG);

/** Background utility for a named tag color, or `undefined` when the name is not a token. */
export function tagColorBg(color?: string | null): string | undefined {
  return color && Object.hasOwn(TAG_COLOR_BG, color) ? TAG_COLOR_BG[color] : undefined;
}
