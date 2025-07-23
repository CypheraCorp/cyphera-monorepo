import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

interface ProductFormData {
  // Product basics
  name: string;
  description?: string;
  external_url?: string;
  
  // Pricing
  price_type: 'recurring' | 'one_time';
  amount: string;
  currency: string;
  interval?: 'day' | 'week' | 'month' | 'year';
  interval_count?: number;
  
  // Payment methods
  selected_networks: { networkId: string; tokenId: string }[];
  
  // Payout
  wallet_id?: string;
  new_wallet_address?: string;
  recipient_network_id?: string;
  recipient_wallet_address?: string;
}

interface CreateProductState {
  // Dialog state
  isOpen: boolean;
  currentStep: number;
  
  // Form data
  formData: Partial<ProductFormData>;
  
  // Validation state
  validatedSteps: Set<number>;
  
  // Loading state
  isCreating: boolean;
  error: string | null;
}

interface CreateProductActions {
  // Dialog actions
  setOpen: (open: boolean) => void;
  setStep: (step: number) => void;
  nextStep: () => void;
  prevStep: () => void;
  
  // Form actions
  updateFormData: (data: Partial<ProductFormData>) => void;
  setFormField: <K extends keyof ProductFormData>(field: K, value: ProductFormData[K]) => void;
  
  // Validation actions
  markStepAsValidated: (step: number) => void;
  clearStepValidation: (step: number) => void;
  
  // Creation actions
  setCreating: (creating: boolean) => void;
  setError: (error: string | null) => void;
  
  // Reset
  reset: () => void;
}

const initialFormData: Partial<ProductFormData> = {
  price_type: 'recurring',
  currency: 'USD',
  interval: 'month',
  interval_count: 1,
  selected_networks: [],
};

const initialState: CreateProductState = {
  isOpen: false,
  currentStep: 0,
  formData: initialFormData,
  validatedSteps: new Set(),
  isCreating: false,
  error: null,
};

export const useCreateProductStore = create<CreateProductState & CreateProductActions>()(
  devtools(
    (set, get) => ({
      ...initialState,

      // Dialog actions
      setOpen: (open) => set({ isOpen: open }),
      setStep: (step) => set({ currentStep: step }),
      nextStep: () => set((state) => ({ 
        currentStep: Math.min(state.currentStep + 1, FORM_STEPS.length - 1) 
      })),
      prevStep: () => set((state) => ({ 
        currentStep: Math.max(state.currentStep - 1, 0) 
      })),
      
      // Form actions
      updateFormData: (data) => set((state) => ({ 
        formData: { ...state.formData, ...data } 
      })),
      setFormField: (field, value) => set((state) => ({
        formData: { ...state.formData, [field]: value }
      })),
      
      // Validation actions
      markStepAsValidated: (step) => set((state) => ({
        validatedSteps: new Set([...state.validatedSteps, step])
      })),
      clearStepValidation: (step) => set((state) => {
        const newSet = new Set(state.validatedSteps);
        newSet.delete(step);
        return { validatedSteps: newSet };
      }),
      
      // Creation actions
      setCreating: (creating) => set({ isCreating: creating }),
      setError: (error) => set({ error }),
      
      // Reset
      reset: () => set({
        ...initialState,
        formData: initialFormData,
        validatedSteps: new Set(),
      }),
    }),
    {
      name: 'create-product-store',
    }
  )
);

// Export the form steps for reuse
export const FORM_STEPS = [
  {
    id: 'basics',
    title: 'Product Basics',
    description: 'Name and description',
  },
  {
    id: 'pricing',
    title: 'Pricing & Billing',
    description: 'Set your price and billing',
  },
  {
    id: 'payment-methods',
    title: 'Payment Options',
    description: 'Choose accepted tokens',
  },
  {
    id: 'payout',
    title: 'Payout Setup',
    description: 'Where you receive payments',
  },
  {
    id: 'review',
    title: 'Review & Create',
    description: 'Confirm and create',
  },
] as const;