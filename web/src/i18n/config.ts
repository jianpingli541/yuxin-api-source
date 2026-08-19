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
import i18n from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

import { convertDetectedLanguage } from './languages'
import en from './locales/en.json'
import zhTW from './locales/zh-TW.json'
import zhCN from './locales/zh.json'

// R3 P2-1: fr/ru/ja/vi 按需懒加载，主 bundle 瘦身约 1.5MB raw
const lazyLocales = {
  fr: () => import('./locales/fr.json'),
  ru: () => import('./locales/ru.json'),
  ja: () => import('./locales/ja.json'),
  vi: () => import('./locales/vi.json'),
} as const

type LazyLng = keyof typeof lazyLocales
const loadedLazy = new Set<string>()

async function ensureLocale(lng: string) {
  const base = lng in lazyLocales ? (lng as LazyLng) : undefined
  if (!base || loadedLazy.has(base)) return
  const mod = await lazyLocales[base]()
  const payload = (mod as { default: { translation: Record<string, unknown> } }).default
  i18n.addResourceBundle(base, 'translation', payload.translation, true, true)
  loadedLazy.add(base)
  if (i18n.language === base) void i18n.changeLanguage(base)
}

export const resources = {
  en,
  zhCN,
  zhTW,
} as const

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: 'en',
    supportedLngs: ['en', 'zhCN', 'fr', 'ru', 'ja', 'vi', 'zhTW'],
    load: 'currentOnly',
    nsSeparator: false, // Allow literal colons in keys (e.g., URLs, labels)
    debug: import.meta.env.DEV,
    interpolation: {
      escapeValue: false, // not needed for react as it escapes by default
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
      // Browsers report `zh-CN`/`zh-TW`/`zh`; map them onto our `zhCN`/`zhTW`
      // codes (non-Chinese codes pass through for normal supportedLngs matching).
      convertDetectedLanguage,
    },
  })

i18n.on('languageChanged', (lng) => {
  void ensureLocale(lng)
})
void ensureLocale(i18n.language)

export default i18n
