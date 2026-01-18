import { useState, useEffect } from 'react'
import QRCode from 'react-qr-code'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

function App() {
  const [status, setStatus] = useState<string>('disconnected')
  const [qrString, setQrString] = useState<string>('')
  const [ws, setWs] = useState<WebSocket | null>(null)

  useEffect(() => {
    const connectWs = () => {
      const newWs = new WebSocket('ws://localhost:8080/ws')
      newWs.onopen = () => {
        console.log('WebSocket connected')
      }
      newWs.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          if (data.type === 'qr') {
            setQrString(data.code)
          } else if (data.type === 'status') {
            setStatus(data.status)
          }
        } catch (err) {
          console.error('Error parsing WebSocket message:', err)
        }
      }
      newWs.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
      newWs.onclose = () => {
        console.log('WebSocket disconnected, reconnecting...')
        setTimeout(connectWs, 1000)
      }
      setWs(newWs)
    }
    connectWs()

    return () => {
      if (ws) ws.close()
    }
  }, [])

  const handleStart = async () => {
    try {
      const response = await fetch('/api/start', { method: 'POST' })
      if (!response.ok) throw new Error('Failed to start')
      console.log('Start command sent')
    } catch (err) {
      console.error('Error starting:', err)
    }
  }

  const handleStop = async () => {
    try {
      const response = await fetch('/api/stop', { method: 'POST' })
      if (!response.ok) throw new Error('Failed to stop')
      console.log('Stop command sent')
    } catch (err) {
      console.error('Error stopping:', err)
    }
  }

  const handleLogout = async () => {
    try {
      const response = await fetch('/api/logout', { method: 'POST' })
      if (!response.ok) throw new Error('Failed to logout')
      console.log('Logout command sent')
    } catch (err) {
      console.error('Error logging out:', err)
    }
  }

  return (
    <div className="container mx-auto p-4">
      <Card>
        <CardHeader>
          <CardTitle>WhatsMeow Control Panel</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <Badge variant={status === 'connected' ? 'default' : 'destructive'}>
              Status: {status}
            </Badge>
          </div>
          {qrString && (
            <div className="flex justify-center">
              <QRCode value={qrString} size={256} />
            </div>
          )}
          <div className="flex space-x-2">
            <Button onClick={handleStart} disabled={status === 'connected'}>
              Start
            </Button>
            <Button onClick={handleStop} variant="outline">
              Stop
            </Button>
            <Button onClick={handleLogout} variant="destructive">
              Logout
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

export default App
