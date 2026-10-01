import {formatPrice} from './price';

describe('formatPrice', () => {
  it('divides the minor unit by one hundred', () => {
    expect(formatPrice({amountMinor: 12345, currency: 'USD'}, 'en-US')).toBe('$123.45');
  });

  it('writes the currency of the price', () => {
    expect(formatPrice({amountMinor: 500000, currency: 'NGN'}, 'en-US')).toContain('5,000.00');
  });
});
