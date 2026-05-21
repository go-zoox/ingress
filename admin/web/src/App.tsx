
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AppLayout } from './layout/AppLayout'
import { ConfigPage } from './pages/ConfigPage'
import { LogsPage } from './pages/LogsPage'
import { OverviewPage } from './pages/OverviewPage'
import { RoutesPage } from './pages/RoutesPage'
import { TLSPage } from './pages/TLSPage'
import { WAFPage } from './pages/WAFPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<OverviewPage />} />
          <Route path="routes" element={<RoutesPage />} />
          <Route path="waf" element={<WAFPage />} />
          <Route path="tls" element={<TLSPage />} />
          <Route path="config" element={<ConfigPage />} />
          <Route path="logs" element={<LogsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
