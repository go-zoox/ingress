
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { OverviewStreamProvider } from './context/OverviewStreamContext'
import { AppLayout } from './layout/AppLayout'
import { CachePage } from './pages/CachePage'
import { ConfigPage } from './pages/ConfigPage'
import { LogsPage } from './pages/LogsPage'
import { OverviewPage } from './pages/OverviewPage'
import { RouteDetailPage } from './pages/RouteDetailPage'
import { RoutesPage } from './pages/RoutesPage'
import { EventsPage } from './pages/EventsPage'
import { InvestigatePage } from './pages/InvestigatePage'
import { SettingsPage } from './pages/SettingsPage'
import { TLSPage } from './pages/TLSPage'
import { TopologyPage } from './pages/TopologyPage'
import { HealthPage } from './pages/HealthPage'
import { WAFPage } from './pages/WAFPage'
import { MaintenancePage } from './pages/MaintenancePage'
import { MessagesPage } from './pages/MessagesPage'

export default function App() {
  return (
    <BrowserRouter>
      <OverviewStreamProvider>
        <Routes>
          <Route element={<AppLayout />}>
          <Route index element={<OverviewPage />} />
          <Route path="attention" element={<Navigate to="/events" replace />} />
          <Route path="routes" element={<RoutesPage />} />
          <Route path="routes/:ruleIndex/:pathIndex" element={<RouteDetailPage />} />
          <Route path="topology" element={<TopologyPage />} />
          <Route path="healths" element={<HealthPage />} />
          <Route path="health" element={<Navigate to="/healths" replace />} />
          <Route path="cache" element={<CachePage />} />
          <Route path="waf" element={<WAFPage />} />
          <Route path="maintenance" element={<MaintenancePage />} />
          <Route path="tls" element={<TLSPage />} />
          <Route path="config" element={<ConfigPage />} />
          <Route path="settings" element={<SettingsPage />} />
          <Route path="logs" element={<LogsPage />} />
          <Route path="events" element={<EventsPage />} />
          <Route path="investigate" element={<InvestigatePage />} />
          <Route path="messages" element={<MessagesPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
      </OverviewStreamProvider>
    </BrowserRouter>
  )
}
