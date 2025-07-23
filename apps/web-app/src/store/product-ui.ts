import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

interface ProductUIState {
  // Selection state
  selectedProductId: string | null;
  
  // View state
  viewMode: 'grid' | 'list';
  
  // Filter state
  filters: {
    status?: 'active' | 'inactive' | 'archived';
    priceType?: 'recurring' | 'one_time';
    search?: string;
  };
  
  // Modal states
  editModalOpen: boolean;
  editingProductId: string | null;
  deleteModalOpen: boolean;
  deletingProductId: string | null;
  duplicateModalOpen: boolean;
  duplicatingProductId: string | null;
  
  // Bulk actions
  selectedProductIds: Set<string>;
  bulkActionMode: boolean;
  
  // Sort state
  sortBy: 'created_at' | 'name' | 'price' | 'subscriptions_count';
  sortOrder: 'asc' | 'desc';
}

interface ProductUIActions {
  // Selection actions
  setSelectedProduct: (id: string | null) => void;
  toggleProductSelection: (id: string) => void;
  selectAllProducts: (ids: string[]) => void;
  clearSelection: () => void;
  
  // View actions
  setViewMode: (mode: ProductUIState['viewMode']) => void;
  
  // Filter actions
  setFilters: (filters: Partial<ProductUIState['filters']>) => void;
  updateFilter: <K extends keyof ProductUIState['filters']>(
    key: K,
    value: ProductUIState['filters'][K]
  ) => void;
  clearFilters: () => void;
  
  // Modal actions
  openEditModal: (productId: string) => void;
  closeEditModal: () => void;
  openDeleteModal: (productId: string) => void;
  closeDeleteModal: () => void;
  openDuplicateModal: (productId: string) => void;
  closeDuplicateModal: () => void;
  
  // Bulk actions
  setBulkActionMode: (enabled: boolean) => void;
  
  // Sort actions
  setSortBy: (sortBy: ProductUIState['sortBy']) => void;
  toggleSortOrder: () => void;
  
  // Reset
  reset: () => void;
}

const initialState: ProductUIState = {
  selectedProductId: null,
  viewMode: 'grid',
  filters: {},
  editModalOpen: false,
  editingProductId: null,
  deleteModalOpen: false,
  deletingProductId: null,
  duplicateModalOpen: false,
  duplicatingProductId: null,
  selectedProductIds: new Set(),
  bulkActionMode: false,
  sortBy: 'created_at',
  sortOrder: 'desc',
};

export const useProductUIStore = create<ProductUIState & ProductUIActions>()(
  devtools(
    (set) => ({
      ...initialState,

      // Selection actions
      setSelectedProduct: (id) => set({ selectedProductId: id }),
      toggleProductSelection: (id) => set((state) => {
        const newSet = new Set(state.selectedProductIds);
        if (newSet.has(id)) {
          newSet.delete(id);
        } else {
          newSet.add(id);
        }
        return { selectedProductIds: newSet };
      }),
      selectAllProducts: (ids) => set({ 
        selectedProductIds: new Set(ids) 
      }),
      clearSelection: () => set({ 
        selectedProductIds: new Set(),
        bulkActionMode: false,
      }),
      
      // View actions
      setViewMode: (mode) => set({ viewMode: mode }),
      
      // Filter actions
      setFilters: (filters) => set((state) => ({
        filters: { ...state.filters, ...filters }
      })),
      updateFilter: (key, value) => set((state) => ({
        filters: { ...state.filters, [key]: value }
      })),
      clearFilters: () => set({ filters: {} }),
      
      // Modal actions
      openEditModal: (productId) => set({
        editModalOpen: true,
        editingProductId: productId,
      }),
      closeEditModal: () => set({
        editModalOpen: false,
        editingProductId: null,
      }),
      openDeleteModal: (productId) => set({
        deleteModalOpen: true,
        deletingProductId: productId,
      }),
      closeDeleteModal: () => set({
        deleteModalOpen: false,
        deletingProductId: null,
      }),
      openDuplicateModal: (productId) => set({
        duplicateModalOpen: true,
        duplicatingProductId: productId,
      }),
      closeDuplicateModal: () => set({
        duplicateModalOpen: false,
        duplicatingProductId: null,
      }),
      
      // Bulk actions
      setBulkActionMode: (enabled) => set({ 
        bulkActionMode: enabled,
        selectedProductIds: enabled ? new Set() : new Set(),
      }),
      
      // Sort actions
      setSortBy: (sortBy) => set({ sortBy }),
      toggleSortOrder: () => set((state) => ({
        sortOrder: state.sortOrder === 'asc' ? 'desc' : 'asc'
      })),
      
      // Reset
      reset: () => set(initialState),
    }),
    {
      name: 'product-ui-store',
    }
  )
);

// Selectors
export const useProductFilters = () => 
  useProductUIStore((state) => state.filters);

export const useSelectedProducts = () => 
  useProductUIStore((state) => state.selectedProductIds);

export const useBulkActionMode = () => 
  useProductUIStore((state) => state.bulkActionMode);