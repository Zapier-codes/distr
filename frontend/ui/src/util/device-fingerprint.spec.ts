import {deviceFingerprint, deviceSignals} from './device-fingerprint';

describe('deviceFingerprint', () => {
  it('is a lower case hex sha-256', async () => {
    expect(await deviceFingerprint(['a', 'b'])).toMatch(/^[0-9a-f]{64}$/);
  });

  it('is stable for the same signals and differs for other signals', async () => {
    expect(await deviceFingerprint(['a', 'b'])).toBe(await deviceFingerprint(['a', 'b']));
    expect(await deviceFingerprint(['a', 'b'])).not.toBe(await deviceFingerprint(['a', 'c']));
  });

  it('does not let two signals run together', async () => {
    expect(await deviceFingerprint(['ab', 'c'])).not.toBe(await deviceFingerprint(['a', 'bc']));
  });
});

describe('deviceSignals (g.i-b: browser-independent, D3)', () => {
  // Two browsers on one device share the hardware and screen signals and differ only in user agent, language and
  // platform. Those browser signals are not in the set, so the two must fingerprint the same.
  const hardware = {
    hardwareConcurrency: 8,
    maxTouchPoints: 0,
    screen: {width: 1920, height: 1080, colorDepth: 24},
    devicePixelRatio: 1,
    timeZone: 'Europe/Berlin',
  };

  it('is identical for two browsers on one device (same hardware and screen)', async () => {
    const chrome = deviceSignals(hardware);
    const firefox = deviceSignals(hardware);
    expect(chrome).toEqual(firefox);
    expect(await deviceFingerprint(chrome)).toBe(await deviceFingerprint(firefox));
  });

  it('never includes a user agent, language or platform', () => {
    const signals = deviceSignals(hardware).join('\u001f');
    expect(signals).not.toMatch(/Mozilla|Chrome|Safari|Firefox|Linux|Windows|Android|Macintosh/);
  });

  it('differs when the hardware or screen differs', async () => {
    const other = {...hardware, screen: {width: 1280, height: 720, colorDepth: 24}};
    expect(await deviceFingerprint(deviceSignals(hardware))).not.toBe(
      await deviceFingerprint(deviceSignals(other))
    );
    const fewerCores = {...hardware, hardwareConcurrency: 4};
    expect(await deviceFingerprint(deviceSignals(hardware))).not.toBe(
      await deviceFingerprint(deviceSignals(fewerCores))
    );
  });

  it('uses five signals, so a missing one still hashes to a stable value', () => {
    expect(deviceSignals({})).toHaveLength(5);
    expect(deviceSignals({}).every((s) => typeof s === 'string')).toBe(true);
  });
});
