import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './index.css'

export type Theme = 'light' | 'dark' | 'system'

const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

export function applyTheme(theme: Theme) {
  const isDark = theme === 'dark' || (theme === 'system' && mediaQuery.matches)
  document.documentElement.classList.toggle('dark', isDark)
}

function initTheme() {
  const stored = localStorage.getItem('theme') as Theme | null
  const theme: Theme = stored === 'light' || stored === 'dark' ? stored : 'system'
  applyTheme(theme)
  mediaQuery.addEventListener('change', () => {
    const current = localStorage.getItem('theme') as Theme | null
    if (!current || current === 'system') applyTheme('system')
  })
}

initTheme()

const container = document.getElementById('root')
const root = createRoot(container!)
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
