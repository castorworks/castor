import * as React from 'react';

const MOBILE_BREAKPOINT = 768;

/** Whether the viewport is narrower than `breakpoint` px (the mobile layout by default). */
export function useIsMobile(breakpoint = MOBILE_BREAKPOINT) {
  const [isMobile, setIsMobile] = React.useState<boolean | undefined>(undefined);

  React.useEffect(() => {
    const mql = window.matchMedia(`(max-width: ${breakpoint - 1}px)`);
    const onChange = () => {
      setIsMobile(window.innerWidth < breakpoint);
    };
    mql.addEventListener('change', onChange);
    setIsMobile(window.innerWidth < breakpoint);
    return () => mql.removeEventListener('change', onChange);
  }, [breakpoint]);

  return !!isMobile;
}
