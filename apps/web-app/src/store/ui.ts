import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';

type Theme = 'light' | 'dark' | 'system';
type UserType = 'merchant' | 'customer' | null;

export interface UIState {
  // Theme preferences
  theme: Theme;

  // Sidebar state
  isSidebarCollapsed: boolean;
  isMobileSidebarOpen: boolean;

  // Modal states
  activeModal: string | null;
  modalData: Record<string, unknown> | null;

  // User type and preferences
  userType: UserType;
  preferredCurrency: string;
  showTestnetBanner: boolean;

  // Loading and progress states
  globalLoading: boolean;
  loadingMessage: string | null;
  progress: number | null;

  // Toast/notification preferences
  enableSoundNotifications: boolean;
  enableDesktopNotifications: boolean;

  // View preferences
  compactView: boolean;
  showAdvancedOptions: boolean;

  // Onboarding state
  hasCompletedOnboarding: boolean;
  currentOnboardingStep: number;
}

export interface UIActions {
  // Theme actions
  setTheme: (theme: Theme) => void;

  // Sidebar actions
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setMobileSidebarOpen: (open: boolean) => void;

  // Modal actions
  openModal: (modalId: string, data?: Record<string, unknown>) => void;
  closeModal: () => void;

  // User preference actions
  setUserType: (type: UserType) => void;
  setPreferredCurrency: (currency: string) => void;
  setShowTestnetBanner: (show: boolean) => void;

  // Loading actions
  setGlobalLoading: (loading: boolean, message?: string) => void;
  setProgress: (progress: number | null) => void;

  // Notification preference actions
  setEnableSoundNotifications: (enable: boolean) => void;
  setEnableDesktopNotifications: (enable: boolean) => void;

  // View preference actions
  setCompactView: (compact: boolean) => void;
  setShowAdvancedOptions: (show: boolean) => void;

  // Onboarding actions
  setHasCompletedOnboarding: (completed: boolean) => void;
  setCurrentOnboardingStep: (step: number) => void;
  advanceOnboardingStep: () => void;

  // Reset actions
  resetUIPreferences: () => void;
}

const initialState: UIState = {
  theme: 'system',
  isSidebarCollapsed: false,
  isMobileSidebarOpen: false,
  activeModal: null,
  modalData: null,
  userType: null,
  preferredCurrency: 'USD',
  showTestnetBanner: true,
  globalLoading: false,
  loadingMessage: null,
  progress: null,
  enableSoundNotifications: false,
  enableDesktopNotifications: false,
  compactView: false,
  showAdvancedOptions: false,
  hasCompletedOnboarding: false,
  currentOnboardingStep: 0,
};

export const useUIStore = create<UIState & UIActions>()(
  devtools(
    persist(
      (set, get) => ({
        ...initialState,

        // Theme actions
        setTheme: (theme) => set({ theme }),

        // Sidebar actions
        toggleSidebar: () => set((state) => ({ isSidebarCollapsed: !state.isSidebarCollapsed })),
        setSidebarCollapsed: (collapsed) => set({ isSidebarCollapsed: collapsed }),
        setMobileSidebarOpen: (open) => set({ isMobileSidebarOpen: open }),

        // Modal actions
        openModal: (modalId, data) => set({ activeModal: modalId, modalData: data || null }),
        closeModal: () => set({ activeModal: null, modalData: null }),

        // User preference actions
        setUserType: (type) => set({ userType: type }),
        setPreferredCurrency: (currency) => set({ preferredCurrency: currency }),
        setShowTestnetBanner: (show) => set({ showTestnetBanner: show }),

        // Loading actions
        setGlobalLoading: (loading, message) =>
          set({ globalLoading: loading, loadingMessage: message || null }),
        setProgress: (progress) => set({ progress }),

        // Notification preference actions
        setEnableSoundNotifications: (enable) => set({ enableSoundNotifications: enable }),
        setEnableDesktopNotifications: (enable) => set({ enableDesktopNotifications: enable }),

        // View preference actions
        setCompactView: (compact) => set({ compactView: compact }),
        setShowAdvancedOptions: (show) => set({ showAdvancedOptions: show }),

        // Onboarding actions
        setHasCompletedOnboarding: (completed) => set({ hasCompletedOnboarding: completed }),
        setCurrentOnboardingStep: (step) => set({ currentOnboardingStep: step }),
        advanceOnboardingStep: () => {
          const { currentOnboardingStep } = get();
          set({ currentOnboardingStep: currentOnboardingStep + 1 });
        },

        // Reset actions
        resetUIPreferences: () =>
          set({
            theme: 'system',
            preferredCurrency: 'USD',
            showTestnetBanner: true,
            enableSoundNotifications: false,
            enableDesktopNotifications: false,
            compactView: false,
            showAdvancedOptions: false,
          }),
      }),
      {
        name: 'ui-storage',
        partialize: (state) => ({
          // Persist user preferences
          theme: state.theme,
          isSidebarCollapsed: state.isSidebarCollapsed,
          userType: state.userType,
          preferredCurrency: state.preferredCurrency,
          showTestnetBanner: state.showTestnetBanner,
          enableSoundNotifications: state.enableSoundNotifications,
          enableDesktopNotifications: state.enableDesktopNotifications,
          compactView: state.compactView,
          showAdvancedOptions: state.showAdvancedOptions,
          hasCompletedOnboarding: state.hasCompletedOnboarding,
          currentOnboardingStep: state.currentOnboardingStep,
        }),
      }
    )
  )
);
