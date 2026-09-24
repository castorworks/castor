import type { IconProps } from '@tabler/icons-react';

// Castor mark: a beaver whose body is the frame, the spiral its coiled back and
// the loop its tail. Drawn with `currentColor`; `src/app/icon.svg` holds the
// same paths for the favicon — keep the two in sync. The stroke width is part
// of the drawing, so Tabler's `stroke` prop is ignored.
export function CastorLogo({ size = 24, title, ...props }: IconProps) {
  return (
    <svg
      {...props}
      xmlns='http://www.w3.org/2000/svg'
      viewBox='0 0 400 400'
      width={size}
      height={size}
      fill='none'
      stroke='currentColor'
      strokeWidth={22}
      strokeLinecap='round'
      strokeLinejoin='round'
      aria-hidden={title ? undefined : true}
    >
      {title && <title>{title}</title>}
      <path d='M204 176a20 20 0 0 1 40 0a42 42 0 0 1-84 0a65 65 0 0 1 130 0a87 82 0 0 1-174 0a100 104 0 0 1 100-104h20a88 88 0 0 1 88 88v46a90 90 0 0 1-90 90H90q-14 0-14-14V60q0-14 14-14h206' />
      <path d='M296 46c-4-18 10-26 24-12c32 2 60 22 68 46c-6 18-30 24-46 28c-12 4-18 16-18 34' />
      <path d='M76 296l-6 14c-26 6-58 20-56 48c2 28 36 34 48 16c10-16 10-40 8-64' />
      <circle cx='352' cy='64' r='5' fill='currentColor' stroke='none' />
    </svg>
  );
}
