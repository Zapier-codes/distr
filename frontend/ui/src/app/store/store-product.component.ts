import {Component, computed, inject, signal, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {catchError, firstValueFrom, map, of, switchMap} from 'rxjs';
import {deviceFingerprint} from '../../util/device-fingerprint';
import {getFormDisplayedError} from '../../util/errors';
import {formatPrice} from '../../util/price';
import {PortalLogoComponent} from '../components/portal-logo/portal-logo.component';
import {TurnstileComponent} from '../components/turnstile.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {PortalService} from '../services/portal.service';
import {ProductRequestsService} from '../services/product-requests.service';
import {ToastService} from '../services/toast.service';
import {ProductRequest, ProductRequestIcon, ProductService} from '../types/product-request';

// The backend enforces the same limit and rejects anything else. Checking here only saves an upload that cannot work.
const maxIconBytes = 256 * 1024;

function toBase64(bytes: Uint8Array): string {
  let binary = '';
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
}

/** One product of the storefront, with its demo material and the form that requests it. */
@Component({
  selector: 'app-store-product',
  imports: [RouterLink, ReactiveFormsModule, AutotrimDirective, PortalLogoComponent, TurnstileComponent],
  templateUrl: './store-product.component.html',
})
export class StoreProductComponent {
  private readonly service = inject(ProductRequestsService);
  private readonly portalService = inject(PortalService);
  private readonly toast = inject(ToastService);

  // undefined while loading, null when the product does not exist or could not be loaded.
  protected readonly product = toSignal<ProductService | null | undefined>(
    inject(ActivatedRoute).paramMap.pipe(
      map((params) => params.get('slug') ?? ''),
      switchMap((slug) => this.service.getProduct(slug).pipe(catchError(() => of(null))))
    ),
    {initialValue: undefined}
  );

  protected readonly screenshots = computed(() => this.product()?.media.filter((m) => m.kind === 'screenshot') ?? []);
  protected readonly videos = computed(() => this.product()?.media.filter((m) => m.kind === 'video') ?? []);

  protected readonly form = new FormGroup({
    appName: new FormControl('', [Validators.required, Validators.maxLength(100)]),
    contactEmail: new FormControl('', [Validators.required, Validators.email, Validators.maxLength(254)]),
    themeColor: new FormControl('#2563eb', [Validators.required, Validators.pattern(/^#[0-9a-fA-F]{6}$/)]),
  });

  protected readonly price = computed(() => {
    const price = this.product()?.price;
    return price ? formatPrice(price) : undefined;
  });

  protected readonly loading = signal(false);
  // The payment page of the submitted request, once it exists.
  protected readonly paymentUrl = signal<string | undefined>(undefined);
  protected readonly submitted = signal<ProductRequest | undefined>(undefined);

  protected readonly icon = signal<ProductRequestIcon | undefined>(undefined);
  protected readonly iconError = signal<string | undefined>(undefined);
  // The icon as a data URL of the PNG the visitor picked, for the preview only.
  protected readonly iconPreview = computed(() => {
    const icon = this.icon();
    return icon ? `data:${icon.contentType};base64,${icon.data}` : undefined;
  });

  protected readonly turnstileSiteKey = computed(() => this.portalService.loginConfig().turnstileSiteKey);
  private readonly turnstile = viewChild(TurnstileComponent);
  private readonly turnstileToken = signal<string | undefined>(undefined);

  protected onTurnstileToken(token: string | undefined): void {
    this.turnstileToken.set(token);
  }

  protected async onIconChange(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.item(0);
    this.icon.set(undefined);
    this.iconError.set(undefined);
    if (!file) {
      return;
    }
    if (file.type !== 'image/png') {
      this.iconError.set('The icon must be a PNG image');
    } else if (file.size > maxIconBytes) {
      this.iconError.set('The icon is too large, the limit is 256 KB');
    } else {
      this.icon.set({contentType: 'image/png', data: toBase64(new Uint8Array(await file.arrayBuffer()))});
    }
    if (this.iconError()) {
      input.value = '';
    }
  }

  protected async submit(): Promise<void> {
    const product = this.product();
    this.form.markAllAsTouched();
    if (!product || !this.form.valid) {
      return;
    }
    const turnstileToken = this.turnstileToken();
    if (this.turnstileSiteKey() && !turnstileToken) {
      this.toast.error(
        this.turnstile()?.unavailable()
          ? 'The challenge that proves you are human could not be loaded. Please reload the page and make sure that challenges.cloudflare.com is not blocked.'
          : 'Please complete the challenge that proves you are human, then try again'
      );
      return;
    }

    this.loading.set(true);
    const value = this.form.getRawValue();
    try {
      const result = await firstValueFrom(
        this.service.submit({
          contactEmail: value.contactEmail!,
          appName: value.appName!,
          productServiceId: product.id,
          themeColor: value.themeColor!,
          icon: this.icon(),
          turnstileToken,
          deviceFingerprint: await deviceFingerprint(),
        })
      );
      this.paymentUrl.set(result.paymentUrl);
      this.submitted.set(result);
    } catch (e) {
      const error = getFormDisplayedError(e);
      if (error) {
        this.toast.error(error);
      }
      // A token can only be redeemed once, so a retry needs a new challenge.
      this.turnstile()?.reset();
    } finally {
      this.loading.set(false);
    }
  }

  /** Asks for the payment page again, for a request whose page could not be made when it was submitted. */
  protected async preparePayment(): Promise<void> {
    const request = this.submitted();
    if (!request) {
      return;
    }
    this.loading.set(true);
    try {
      this.paymentUrl.set((await firstValueFrom(this.service.payment(request.id))).paymentUrl);
    } catch (e) {
      const error = getFormDisplayedError(e);
      if (error) {
        this.toast.error(error);
      }
    } finally {
      this.loading.set(false);
    }
  }
}
