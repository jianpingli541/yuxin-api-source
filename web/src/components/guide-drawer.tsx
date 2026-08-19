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
import { BookOpen } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Markdown } from '@/components/ui/markdown'
import { useIsAdmin } from '@/hooks/use-admin'
import { adminHelpMd } from '@/routes/_authenticated/admin-help/index'
import { userHelpMd } from '@/routes/help/index'
import { useGuideDrawerStore } from '@/stores/guide-drawer-store'

/**
 * In-page guide drawer. Opens from the sidebar '使用指南' entry and renders
 * the role-appropriate guide inline (user guide for normal users, admin guide
 * for admins) without navigating away from the current page.
 */
export function GuideDrawer() {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const open = useGuideDrawerStore((state) => state.open)
  const setOpen = useGuideDrawerStore((state) => state.setOpen)

  const title = isAdmin ? t('Admin Guide') : t('User Guide')
  const content = isAdmin ? adminHelpMd : userHelpMd

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetContent
        side='right'
        className={sideDrawerContentClassName('sm:max-w-2xl')}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle className='flex items-center gap-2'>
            <BookOpen className='size-4 shrink-0' />
            {title}
          </SheetTitle>
          <SheetDescription className='sr-only'>
            {title}
          </SheetDescription>
        </SheetHeader>
        <div className={sideDrawerFormClassName()}>
          <Markdown>{content}</Markdown>
        </div>
      </SheetContent>
    </Sheet>
  )
}
