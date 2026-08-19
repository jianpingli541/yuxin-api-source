/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { create } from 'zustand'

/**
 * Global state for the in-page guide drawer. The sidebar '使用指南' entry
 * opens this drawer (instead of navigating to /help), and the drawer renders
 * the role-appropriate guide inline.
 */
interface GuideDrawerState {
  open: boolean
  setOpen: (open: boolean) => void
}

export const useGuideDrawerStore = create<GuideDrawerState>()((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}))

export function openGuideDrawer(): void {
  useGuideDrawerStore.getState().setOpen(true)
}
