/**
 * The device signals the free-tier check uses: stable, browser-independent properties of the hardware and the
 * screen, nothing that identifies the visitor. They are hashed here, so only the digest leaves the browser, and the
 * server keys that digest with a secret before it stores anything. Used for nothing but giving each device one free
 * product.
 *
 * Slice g.i-b (D3): the browser-specific signals (user agent, language, platform) are deliberately NOT used. They
 * differ between two browsers on one device, which used to let the same person claim a second free product. The
 * graphics-card renderer string is not used either: D3 allows it "only if it proves stable", and stability was not
 * measured on real devices, while an unstable renderer string would split one device's hash in two and hand out a
 * second free product. Kept to hardware and screen signals, so two browsers on one device agree.
 */

/** The stable signals, in a fixed order. `deviceSignals` takes them from here so a spec can prove the exact set. */
export interface DeviceSignalSource {
  hardwareConcurrency?: number;
  maxTouchPoints?: number;
  screen?: { width?: number; height?: number; colorDepth?: number };
  devicePixelRatio?: number;
  timeZone?: string;
}

function resolvedTimeZone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone ?? '';
  } catch {
    return '';
  }
}

function browserSignals(): DeviceSignalSource {
  const nav = typeof navigator === 'undefined' ? undefined : navigator;
  const scr = typeof screen === 'undefined' ? undefined : screen;
  return {
    hardwareConcurrency: nav?.hardwareConcurrency,
    maxTouchPoints: nav?.maxTouchPoints,
    screen: scr ? { width: scr.width, height: scr.height, colorDepth: scr.colorDepth } : undefined,
    devicePixelRatio: typeof window === 'undefined' ? undefined : window.devicePixelRatio,
    timeZone: resolvedTimeZone(),
  };
}

export function deviceSignals(source: DeviceSignalSource = browserSignals()): string[] {
  const scr = source.screen;
  return [
    String(source.hardwareConcurrency ?? ''),
    String(source.maxTouchPoints ?? ''),
    scr ? `${scr.width ?? ''}x${scr.height ?? ''}x${scr.colorDepth ?? ''}` : '',
    String(source.devicePixelRatio ?? ''),
    source.timeZone ?? '',
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
