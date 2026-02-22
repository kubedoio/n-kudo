import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export async function copyToClipboard(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    const textArea = document.createElement('textarea')
    textArea.value = text
    textArea.style.position = 'fixed'
    textArea.style.left = '-999999px'
    document.body.appendChild(textArea)
    textArea.select()
    document.execCommand('copy')
    document.body.removeChild(textArea)
  }
}

export function formatRelativeTime(date: string | Date): string {
  const now = new Date()
  const then = typeof date === 'string' ? new Date(date) : date
  const diffMs = now.getTime() - then.getTime()
  const diffSecs = Math.floor(diffMs / 1000)
  const diffMins = Math.floor(diffSecs / 60)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffSecs < 60) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`
  return then.toLocaleDateString()
}

export function truncateId(id: string, length = 8): string {
  if (id.length <= length) return id
  return id.slice(0, length)
}

export function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}
