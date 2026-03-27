import {
  ReactNode,
  useState,
  createContext,
  useContext,
  useEffect,
  useCallback,
} from "react"
import { useKaradulStore } from "@/lib/store"

interface WebSocketContextType {
  connected: boolean
  error: string | null
}

const WebSocketContext = createContext<WebSocketContextType>({
  connected: false,
  error: null,
})

export function useWebSocket() {
  return useContext(WebSocketContext)
}

interface WebSocketProviderProps {
  children: ReactNode
  url?: string
}

export function WebSocketProvider({
  children,
  url = `ws://${window.location.host}/ws`,
}: WebSocketProviderProps) {
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const setNodes = useKaradulStore((state) => state.setNodes)
  const setPeers = useKaradulStore((state) => state.setPeers)
  const setTopology = useKaradulStore((state) => state.setTopology)
  const setStats = useKaradulStore((state) => state.setStats)

  const connect = useCallback(() => {
    try {
      const ws = new WebSocket(url)

      ws.onopen = () => {
        setConnected(true)
        setError(null)
      }

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data)

          switch (message.type) {
            case "nodes":
              setNodes(message.data)
              break
            case "peers":
              setPeers(message.data)
              break
            case "topology":
              setTopology(message.data)
              break
            case "stats":
              setStats(message.data)
              break
            default:
              console.warn("Unknown message type:", message.type)
          }
        } catch (err) {
          console.error("Failed to parse WebSocket message:", err)
        }
      }

      ws.onclose = () => {
        setConnected(false)
        // Auto-reconnect after 3 seconds
        setTimeout(connect, 3000)
      }

      ws.onerror = () => {
        setError("WebSocket connection failed")
        setConnected(false)
        ws.close()
      }

      return () => {
        ws.close()
      }
    } catch {
      setError("Failed to connect to WebSocket")
      setTimeout(connect, 3000)
    }
  }, [url, setNodes, setPeers, setTopology, setStats])

  useEffect(() => {
    const cleanup = connect()
    return () => {
      if (cleanup) cleanup()
    }
  }, [connect])

  return (
    <WebSocketContext.Provider value={{ connected, error }}>
      {children}
    </WebSocketContext.Provider>
  )
}
