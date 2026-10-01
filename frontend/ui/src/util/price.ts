import {ProductPrice} from '../app/types/product-request';

/**
 * A one-time price as text, in the currency's own format. The amount is in the minor unit, which is a hundredth for
 * every currency the storefront sells in (a currency with another minor unit, such as the yen, is not supported).
 */
export function formatPrice(price: ProductPrice, locale?: string): string {
  return new Intl.NumberFormat(locale, {style: 'currency', currency: price.currency}).format(price.amountMinor / 100);
}
