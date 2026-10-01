import {deviceFingerprint} from './device-fingerprint';

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
