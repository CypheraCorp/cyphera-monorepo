'use client';

import { useState, useEffect } from 'react';
import { notFound } from 'next/navigation';
import Image from 'next/image';
import Link from 'next/link';
import { ProductPaymentCard } from '@/components/public/product-payment-card';
import { PublicHeader } from '@/components/public/public-header';
import { USDCBalanceCard } from '@/components/public/usdc-balance-card';
import { PublicProductResponse } from '@/types/product';
import { useWeb3AuthInitialization } from '@/hooks/auth';
import { logger } from '@/lib/core/logger/logger-utils';
import { ProductCardSkeleton, BalanceCardSkeleton } from '@/components/ui/loading-states';
import { Skeleton } from '@/components/ui/skeleton';

// Component displays a product page using the productId route parameter
interface PayProductPageProps {
  params: Promise<{ productId: string }>;
}

export default function PayProductPage({ params }: PayProductPageProps) {
  const [product, setProduct] = useState<PublicProductResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [resolvedParams, setResolvedParams] = useState<{ productId: string } | null>(
    null
  );

  // Web3Auth initialization tracking
  const { isInitializing, isAuthenticated, isConnected } = useWeb3AuthInitialization();

  // Resolve params first
  useEffect(() => {
    async function resolveParams() {
      try {
        const resolved = await params;
        setResolvedParams(resolved);
      } catch (error) {
        logger.error('Failed to resolve params', error);
        setError('Failed to load page parameters');
        setLoading(false);
      }
    }
    resolveParams();
  }, [params]);

  // Fetch product data once params are resolved
  useEffect(() => {
    if (!resolvedParams?.productId) return;

    async function fetchProduct() {
      try {
        setLoading(true);
        setError(null);

        // Type guard to ensure resolvedParams is not null
        if (!resolvedParams) return;

        logger.debug('Fetching product for productId:', { productId: resolvedParams.productId });

        const response = await fetch(`/api/pay/${resolvedParams.productId}`);
        if (!response.ok) {
          const errorData = await response.json();
          throw new Error(errorData.error || 'Failed to fetch product');
        }

        const productData = await response.json();
        logger.debug('Product data received', { productData });
        setProduct(productData);
      } catch (err) {
        logger.error('Failed to fetch product', err);

        // More detailed error handling
        if (err instanceof Error) {
          setError(err.message);
        } else {
          setError('Failed to load product');
        }
      } finally {
        setLoading(false);
      }
    }

    fetchProduct();
  }, [resolvedParams]);

  // Force re-render when authentication state changes
  useEffect(() => {
    logger.debug('Authentication state changed', { isAuthenticated, isConnected });
  }, [isAuthenticated, isConnected]);

  // Show loading state for product data OR Web3Auth initialization
  if (loading || isInitializing) {
    return (
      <div className="min-h-screen bg-neutral-50 dark:bg-neutral-900">
        <PublicHeader />

        <div className="container mx-auto p-8 space-y-8">
          <div className="max-w-4xl mx-auto">
            {/* Product Header Skeleton */}
            <div className="text-center mb-8">
              <Skeleton className="h-12 w-96 mx-auto mb-4" />
              <Skeleton className="h-6 w-full max-w-2xl mx-auto mb-2" />
              <Skeleton className="h-6 w-3/4 max-w-2xl mx-auto mb-6" />

              {/* Product Image Skeleton */}
              <Skeleton className="w-full max-w-2xl mx-auto h-64 rounded-lg mb-8" />
            </div>

            {/* Payment Card Skeleton */}
            <div className="text-center space-y-6">
              <div className="max-w-md mx-auto">
                <ProductCardSkeleton />
              </div>

              {/* Balance Card Skeleton - Only show when we expect authentication */}
              {!loading && isInitializing && (
                <div className="max-w-md mx-auto">
                  <BalanceCardSkeleton />
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 container mx-auto p-8 space-y-8">
        <div className="text-center">
          <h1 className="text-4xl font-bold mb-4">Error Loading Product</h1>
          <p className="text-red-600 mb-4">{error}</p>
          <Link
            href="/"
            className="inline-block px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700"
          >
            Return Home
          </Link>
        </div>
      </div>
    );
  }

  if (!product) {
    notFound();
  }

  // Always show product information publicly
  return (
    <div className="min-h-screen bg-neutral-50 dark:bg-neutral-900">
      <PublicHeader />

      <div className="container mx-auto p-8 space-y-8">
        <div className="max-w-4xl mx-auto">
          {/* Product Header */}
          <div className="text-center mb-8">
            <h1 className="text-5xl font-bold mb-4">{product.name || 'Product'}</h1>
            {product.description && (
              <p className="text-xl text-muted-foreground mb-6">{product.description}</p>
            )}

            {/* Product Image */}
            {product.image_url && product.image_url.trim() !== '' && (
              <div className="relative w-full max-w-2xl mx-auto h-64 rounded-lg overflow-hidden mb-8">
                <Image
                  src={product.image_url}
                  alt={product.name || 'Product'}
                  fill
                  className="object-cover"
                  priority
                />
              </div>
            )}
          </div>

          {/* Call to Action */}
          <div className="text-center space-y-6">
            <div className="max-w-md mx-auto">
              <ProductPaymentCard
                key={`payment-card-${isAuthenticated}-${isConnected}`}
                product={product}
                isAuthenticated={isAuthenticated}
              />
            </div>

            {/* USDC Balance & Faucet - Only show when authenticated */}
            {isAuthenticated && (
              <div className="max-w-md mx-auto">
                <USDCBalanceCard productNetwork={product.product_tokens?.[0]} />
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}