import { create } from 'zustand'

interface Tenant {
  id: string
  name: string
}

interface AppState {
  currentTenant: Tenant | null
  setCurrentTenant: (tenant: Tenant | null) => void
  sidebarOpen: boolean
  toggleSidebar: () => void
}

export const useAppStore = create<AppState>((set) => ({
  currentTenant: null,
  setCurrentTenant: (tenant) => set({ currentTenant: tenant }),
  sidebarOpen: true,
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
}))
