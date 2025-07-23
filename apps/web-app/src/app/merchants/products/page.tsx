'use client';

import { Button } from '@/components/ui/button';
import { formatProductInterval, formatProductType } from '@/lib/features/products/products';
import { SUPPORTED_CURRENCIES, DEFAULT_CURRENCY, CURRENCY_SYMBOLS } from '@/lib/constants/currency';
import { ProductTokenResponse } from '@/types/product';
import { TokenResponse } from '@/types/token';
import { useSearchParams } from 'next/navigation';
import { useProductsPageData } from '@/hooks/data';
import { ProductCardSkeleton } from '@/components/ui/loading-states';
import { Suspense } from 'react';
import dynamic from 'next/dynamic';

// Dynamically import lucide-react icons
const Package = dynamic(() => import('lucide-react').then((mod) => ({ default: mod.Package })), {
  loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded-sm" />,
  ssr: false,
});

const MoreHorizontal = dynamic(
  () => import('lucide-react').then((mod) => ({ default: mod.MoreHorizontal })),
  {
    loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded-sm" />,
    ssr: false,
  }
);

const Copy = dynamic(() => import('lucide-react').then((mod) => ({ default: mod.Copy })), {
  loading: () => <div className="h-4 w-4 bg-muted animate-pulse rounded-sm" />,
  ssr: false,
});

// Dynamically import heavy components to reduce initial bundle size
const DropdownMenu = dynamic(
  () => import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenu })),
  {
    loading: () => <div className="h-8 w-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const DropdownMenuContent = dynamic(
  () =>
    import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenuContent })),
  {
    loading: () => <div className="h-20 w-32 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const DropdownMenuTrigger = dynamic(
  () =>
    import('@/components/ui/dropdown-menu').then((mod) => ({ default: mod.DropdownMenuTrigger })),
  {
    loading: () => <div className="h-8 w-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const CreateProductDialog = dynamic(
  () =>
    import('@/components/products/create-product-multi-step-dialog').then((mod) => ({
      default: mod.CreateProductMultiStepDialog,
    })),
  {
    loading: () => <div className="h-10 w-32 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const DeleteProductButton = dynamic(
  () =>
    import('@/components/products/delete-product-button').then((mod) => ({
      default: mod.DeleteProductButton,
    })),
  {
    loading: () => <div className="h-8 w-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

// PublishProductDialog was removed - functionality moved elsewhere

const ProductsPagination = dynamic(
  () =>
    import('@/components/products/products-pagination').then((mod) => ({
      default: mod.ProductsPagination,
    })),
  {
    loading: () => <div className="h-10 w-full bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const ProductsRefreshHandler = dynamic(
  () =>
    import('@/components/products/products-refresh-handler').then((mod) => ({
      default: mod.ProductsRefreshHandler,
    })),
  {
    loading: () => <div className="h-8 w-8 bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

const ITEMS_PER_PAGE = 10;

export default function ProductsPage() {
  const searchParams = useSearchParams();
  const currentPage = Number(searchParams.get('page')) || 1;

  // Use React Query hook for cached data fetching
  const {
    products: productsData,
    networks: activeNetworks,
    wallets,
    isLoading: loading,
    error,
  } = useProductsPageData(currentPage, ITEMS_PER_PAGE);

  if (loading) {
    return (
      <div className="space-y-6">
        <Suspense fallback={<div className="h-8 w-8 bg-muted animate-pulse rounded-md" />}>
          <ProductsRefreshHandler />
        </Suspense>
        <div className="flex justify-between items-center">
          <div className="h-10 w-32 bg-muted animate-pulse rounded-md" />
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <ProductCardSkeleton key={i} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="text-red-500">Error: {error.message}</div>
      </div>
    );
  }

  const products = productsData?.data || [];
  const safeActiveNetworks = activeNetworks || [];
  const safeWallets = wallets || [];

  return (
    <div className="flex flex-col min-h-[calc(100vh-200px)]">
      <div className="flex-1 space-y-6">
        <Suspense fallback={<div className="h-8 w-8 bg-muted animate-pulse rounded-md" />}>
          <ProductsRefreshHandler />
        </Suspense>

        <div className="flex justify-end">
          <Suspense fallback={<div className="h-10 w-32 bg-muted animate-pulse rounded-md" />}>
            <CreateProductDialog
              trigger={<Button className="flex items-center gap-2">Create Product</Button>}
              networks={safeActiveNetworks}
              wallets={safeWallets}
              supportedCurrencies={SUPPORTED_CURRENCIES}
              defaultCurrency={DEFAULT_CURRENCY}
            />
          </Suspense>
        </div>

        {products.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 border rounded-lg bg-white dark:bg-neutral-900">
            <Suspense fallback={<div className="h-12 w-12 bg-muted animate-pulse rounded-md" />}>
              <Package className="h-12 w-12 text-muted-foreground mb-4" />
            </Suspense>
            <p className="text-muted-foreground text-center mb-4">
              No products found. Add your first product to get started.
            </p>
            <Suspense fallback={<div className="h-10 w-32 bg-muted animate-pulse rounded-md" />}>
              <CreateProductDialog
                trigger={<Button className="flex items-center gap-2">Create Product</Button>}
                networks={safeActiveNetworks}
                wallets={safeWallets}
                supportedCurrencies={SUPPORTED_CURRENCIES}
                defaultCurrency={DEFAULT_CURRENCY}
              />
            </Suspense>
          </div>
        ) : (
          <>
            <div className="bg-white dark:bg-neutral-900 border border-gray-200 dark:border-gray-800 rounded-lg overflow-hidden">
              {/* Table Header */}
              <div className="bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-3">
                <div className="grid grid-cols-12 gap-4 text-sm font-medium text-gray-700 dark:text-gray-300">
                  <div className="col-span-3">Product</div>
                  <div className="col-span-2">Type & Billing</div>
                  <div className="col-span-1">Price</div>
                  <div className="col-span-1">Status</div>
                  <div className="col-span-3">Payment Methods</div>
                  <div className="col-span-2">Actions</div>
                </div>
              </div>

              {/* Product Rows */}
              <div className="divide-y divide-gray-200 dark:divide-gray-700">
                {products.map((product) => (
                  <Suspense
                    key={product.id}
                    fallback={<div className="h-20 bg-muted animate-pulse" />}
                  >
                    <div className="px-6 py-4 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
                      <div className="grid grid-cols-12 gap-4 items-center">
                        {/* Product Info */}
                        <div className="col-span-3">
                          <div className="space-y-1">
                            <h3 className="font-semibold text-lg text-gray-900 dark:text-gray-100">
                              {product.name}
                            </h3>
                            {product.description && (
                              <p className="text-sm text-muted-foreground line-clamp-2">
                                {product.description}
                              </p>
                            )}
                          </div>
                        </div>

                        {/* Type & Billing */}
                        <div className="col-span-2">
                          <div className="space-y-1">
                            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                              {formatProductType(product.prices?.[0]?.type || '')}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {formatProductInterval(product)}
                            </p>
                          </div>
                        </div>

                        {/* Price */}
                        <div className="col-span-1">
                          <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                            {product.prices?.[0]?.currency === 'EUR' ? (
                              <>
                                {((product.prices?.[0]?.unit_amount_in_pennies ?? 0) / 100).toFixed(
                                  2
                                )}
                                {
                                  CURRENCY_SYMBOLS[
                                    product.prices?.[0]?.currency as keyof typeof CURRENCY_SYMBOLS
                                  ]
                                }
                              </>
                            ) : (
                              <>
                                {
                                  CURRENCY_SYMBOLS[
                                    product.prices?.[0]?.currency as keyof typeof CURRENCY_SYMBOLS
                                  ]
                                }
                                {((product.prices?.[0]?.unit_amount_in_pennies ?? 0) / 100).toFixed(
                                  2
                                )}
                              </>
                            )}
                          </p>
                        </div>

                        {/* Status */}
                        <div className="col-span-1">
                          {product.active ? (
                            <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-300">
                              Active
                            </span>
                          ) : (
                            <span className="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-800 dark:bg-gray-700 dark:text-gray-300">
                              Draft
                            </span>
                          )}
                        </div>

                        {/* Payment Methods */}
                        <div className="col-span-3">
                          <div className="flex items-center gap-1 flex-wrap">
                            {product.product_tokens
                              ?.filter(
                                (token: ProductTokenResponse) =>
                                  token.id && token.network_id && token.token_id && token.active
                              )
                              .map((token: ProductTokenResponse) => {
                                const network = safeActiveNetworks.find(
                                  (n) => n.network.id === token.network_id
                                );
                                const tokenInfo = network?.tokens.find(
                                  (t: TokenResponse) => t.id === token.token_id
                                );
                                if (!network || !tokenInfo) return null;

                                return (
                                  <span
                                    key={`${token.network_id}:${token.token_id}`}
                                    className="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-300"
                                  >
                                    {network.network.name} {tokenInfo.symbol}
                                  </span>
                                );
                              })}
                          </div>
                        </div>

                        {/* Actions */}
                        <div className="col-span-2">
                          <div className="flex items-center gap-2">
                            {/* Product Link for Active Products */}
                            {product.active && (
                              <div className="flex items-center gap-1">
                                <a
                                  href={`/public/prices/${product.prices?.[0]?.id}`}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-200 text-xs font-medium"
                                >
                                  Product Link
                                </a>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="h-5 w-5 p-0"
                                  onClick={() => {
                                    const link = `${window.location.origin}/public/prices/${product.prices?.[0]?.id}`;
                                    navigator.clipboard.writeText(link);
                                  }}
                                >
                                  <Suspense
                                    fallback={
                                      <div className="h-3 w-3 bg-muted animate-pulse rounded-sm" />
                                    }
                                  >
                                    <Copy className="h-3 w-3" />
                                  </Suspense>
                                </Button>
                              </div>
                            )}

                            {/* PublishProductDialog removed - publish functionality moved elsewhere */}

                            {/* Dropdown Menu */}
                            <Suspense
                              fallback={
                                <div className="h-8 w-8 bg-muted animate-pulse rounded-md" />
                              }
                            >
                              <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                  <Button variant="ghost" size="icon" className="h-8 w-8">
                                    <MoreHorizontal className="h-4 w-4" />
                                  </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end">
                                  <DeleteProductButton productId={product.id} />
                                </DropdownMenuContent>
                              </DropdownMenu>
                            </Suspense>
                          </div>
                        </div>
                      </div>
                    </div>
                  </Suspense>
                ))}
              </div>
            </div>
          </>
        )}
      </div>

      {productsData && (
        <div className="mt-8 pt-4">
          <Suspense fallback={<div className="h-10 w-full bg-muted animate-pulse rounded-md" />}>
            <ProductsPagination pageData={productsData} />
          </Suspense>
        </div>
      )}
    </div>
  );
}
