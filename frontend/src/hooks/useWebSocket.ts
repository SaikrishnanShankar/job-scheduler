import { useEffect, useRef, useCallback } from 'react'
import { WSMessage } from '../types'

type MessageHandler = (msg: WSMessage) => void

export function useWebSocket(onMessage: MessageHandler) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const onMessageRef = useRef(onMessage)
  onMessageRef.current = onMessage

  const connect = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const url = `${protocol}//${host}/ws`

    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      console.log('[ws] connected')
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
        reconnectTimerRef.current = null
      }
    }

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)
        onMessageRef.current(msg)
      } catch (e) {
        console.error('[ws] parse error', e)
      }
    }

    ws.onclose = () => {
      console.log('[ws] disconnected — reconnecting in 3s')
      reconnectTimerRef.current = setTimeout(connect, 3000)
    }

    ws.onerror = (err) => {
      console.error('[ws] error', err)
      ws.close()
    }
  }, [])

  useEffect(() => {
    connect()
    return () => {
      if (wsRef.current) wsRef.current.close()
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
    }
  }, [connect])
}
