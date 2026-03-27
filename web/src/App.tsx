import { BrowserRouter, Routes, Route } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ThemeProvider } from "@/components/theme-provider"
import { WebSocketProvider } from "@/lib/websocket"
import { Layout } from "@/components/layout"
import { DashboardPage } from "@/pages/dashboard"
import { TopologyPage } from "@/pages/topology"
import { NodesPage } from "@/pages/nodes"
import { PeersPage } from "@/pages/peers"
import { SettingsPage } from "@/pages/settings"
import { NotFoundPage } from "@/pages/not-found"
import { Toaster } from "@/components/ui/sonner"
import "./index.css"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5000,
      retry: 3,
    },
  },
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider defaultTheme="system" storageKey="karadul-theme">
        <WebSocketProvider>
          <BrowserRouter>
            <Routes>
              <Route path="/" element={<Layout />}>
                <Route index element={<DashboardPage />} />
                <Route path="topology" element={<TopologyPage />} />
                <Route path="nodes" element={<NodesPage />} />
                <Route path="peers" element={<PeersPage />} />
                <Route path="settings" element={<SettingsPage />} />
              </Route>
              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </BrowserRouter>
          <Toaster />
        </WebSocketProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

export default App
