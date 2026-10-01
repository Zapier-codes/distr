import {formatPrice} from '../../util/price';
import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faGlobe, faMobileScreen} from '@fortawesome/free-solid-svg-icons';
import {catchError, of} from 'rxjs';
import {PortalLogoComponent} from '../components/portal-logo/portal-logo.component';
import {ProductRequestsService} from '../services/product-requests.service';
import {ProductService} from '../types/product-request';

interface ProductCard {
  product: ProductService;
  cover?: string;
  price?: string;
}

/** The public storefront: every product a visitor can request, without an account. */
@Component({
  selector: 'app-store',
  imports: [RouterLink, FaIconComponent, PortalLogoComponent],
  templateUrl: './store.component.html',
})
export class StoreComponent {
  protected readonly faMobileScreen = faMobileScreen;
  protected readonly faGlobe = faGlobe;

  // undefined while loading, null when the catalog could not be loaded.
  private readonly products = toSignal(
    inject(ProductRequestsService)
      .listProducts()
      .pipe(catchError(() => of(null))),
    {initialValue: undefined}
  );

  protected readonly cards = computed<ProductCard[] | null | undefined>(() => {
    const products = this.products();
    // Keep null (failed) and undefined (loading) apart from a list.
    return products
      ? products.map((product) => ({
          product,
          cover: product.media.find((m) => m.kind === 'screenshot')?.url,
          price: product.price ? formatPrice(product.price) : undefined,
        }))
      : products;
  });
}
