export type ProductServiceType = 'app' | 'website';

export type ProductServiceMediaKind = 'screenshot' | 'video';

export interface ProductServiceMediaItem {
  kind: ProductServiceMediaKind;
  url: string;
  caption?: string;
}

export interface ProductService {
  id: string;
  slug: string;
  type: ProductServiceType;
  name: string;
  summary: string;
  description: string;
  media: ProductServiceMediaItem[];
}

export interface ProductRequestIcon {
  contentType: 'image/png';
  // Base64 of the PNG bytes.
  data: string;
}

export interface CreateProductRequestRequest {
  contactEmail: string;
  appName: string;
  productServiceId: string;
  themeColor: string;
  icon?: ProductRequestIcon;
  turnstileToken?: string;
  // SHA-256 of device signals, hex. Only used to give each device one free product.
  deviceFingerprint?: string;
}

export type TenantBuildStatus = 'awaiting_gate' | 'queued' | 'building' | 'succeeded' | 'failed';

export type ProductRequestGate = 'free' | 'payment_required';

export interface ProductRequest {
  id: string;
  createdAt: string;
  tenantId: string;
  buildStatus: TenantBuildStatus;
  gate: ProductRequestGate;
}
