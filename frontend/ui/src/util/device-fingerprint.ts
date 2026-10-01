/**
 * The device signals the free-tier check uses: stable properties of the browser and the screen, nothing that
 * identifies the visitor. They are hashed here, so only the digest leaves the browser, and the server keys that digest
 * with a secret before it stores anything. Used for nothing but giving each device one free product.
 */
export function deviceSignals(): string[] {
  const nav = typeof navigator === 'undefined' ? undefined : navigator;
  const scr = typeof screen === 'undefined' ? undefined : screen;
  let timeZone = '';
  try {
    timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone ?? '';
  } catch {
    timeZone = '';
  }
  return [
    nav?.userAgent ?? '',
    nav?.language ?? '',
    String(nav?.hardwareConcurrency ?? ''),
    String(nav?.maxTouchPoints ?? ''),
    nav?.platform ?? '',
    scr ? `${scr.width}x${scr.height}x${scr.colorDepth}` : '',
    String(typeof window === 'undefined' ? '' : window.devicePixelRatio),
    timeZone,
  ];
}

/** The lower case hex SHA-256 of the signals, or undefined when the browser cannot compute it. */
export async function deviceFingerprint(signals: string[] = deviceSignals()): Promise<string | undefined> {
  if (typeof crypto === 'undefined' || !crypto.subtle) {
    return undefined;
  }
  const bytes = new TextEncoder().encode(signals.join('\u001f'));
  const digest = await crypto.subtle.digest('SHA-256', bytes);
  return Array.from(new Uint8Array(digest), (b) => b.toString(16).padStart(2, '0')).join('');
}
