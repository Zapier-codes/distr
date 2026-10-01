import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {CreateProductRequestRequest, ProductRequest, ProductRequestPayment, ProductService} from '../types/product-request';

const publicBaseUrl = '/api/public/v1';

/**
 * The public, no-account request flow: the storefront catalog and the submission of a request. None of these calls
 * carries a credential.
 */
@Injectable({providedIn: 'root'})
export class ProductRequestsService {
  private readonly httpClient = inject(HttpClient);

  public listProducts(): Observable<ProductService[]> {
    return this.httpClient.get<ProductService[]>(`${publicBaseUrl}/product-services`);
  }

  public getProduct(slug: string): Observable<ProductService> {
    return this.httpClient.get<ProductService>(`${publicBaseUrl}/product-services/${encodeURIComponent(slug)}`);
  }

  public submit(request: CreateProductRequestRequest): Observable<ProductRequest> {
    return this.httpClient.post<ProductRequest>(`${publicBaseUrl}/product-requests`, request);
  }

  /** The payment page of a request that has to be paid for. The request id is the reference of the submission. */
  public payment(requestId: string): Observable<ProductRequestPayment> {
    return this.httpClient.post<ProductRequestPayment>(
      `${publicBaseUrl}/product-requests/${encodeURIComponent(requestId)}/payment`,
      {}
    );
  }
}
