
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AppLayout } from './layout/AppLayout'
import { CachePage } from './pages/CachePage'
import { ConfigPage } from './pages/ConfigPage'
import { LogsPage } from './pages/LogsPage'
import { OverviewPage } from './pages/OverviewPage'
import { RoutesPage } from './pages/RoutesPage'
import { SettingsPage } from './pages/SettingsPage'
import { TLSPage } from './pages/TLSPage'
import { WAFPage } from './pages/WAFPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<OverviewPage />} />
          <Route path="routes" element={<RoutesPage />} />
          <Route path="cache" element={<CachePage />} />
          <Route path="waf" element={<WAFPage />} />
          <Route path="tls" element={<TLSPage />} />
          <Route path="config" element={<ConfigPage />} />
          <Route path="settings" element={<SettingsPage />} />
          <Route path="logs" element={<LogsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
