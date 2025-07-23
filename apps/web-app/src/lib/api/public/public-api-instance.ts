import { PublicAPI } from '@/services/cyphera-api/public';

let publicApiInstance: PublicAPI | null = null;

export function getPublicAPI(): PublicAPI {
  if (!publicApiInstance) {
    publicApiInstance = new PublicAPI();
  }
  return publicApiInstance;
}
