import { useLocation } from 'react-router-dom'
import { useEffect, useState } from 'react'

interface PageTransitionProps {
  children: React.ReactNode
}

export function PageTransition({ children }: PageTransitionProps) {
  const location = useLocation()
  const [visible, setVisible] = useState(true)

  useEffect(() => {
    setVisible(false)
    const t = setTimeout(() => setVisible(true), 50)
    return () => clearTimeout(t)
  }, [location.pathname])

  return (
    <div
      className={`transition-opacity duration-150 ${visible ? 'opacity-100' : 'opacity-0'}`}
    >
      {children}
    </div>
  )
}
