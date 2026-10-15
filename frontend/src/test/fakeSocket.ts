import { vi } from 'vitest'

/** Подменяет WebSocket: тест сам открывает, закрывает соединение и шлёт сообщения. */
export class FakeSocket {
  static instances: FakeSocket[] = []
  onopen: (() => void) | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  closed = false
  url: string

  constructor(url: string) {
    this.url = url
    FakeSocket.instances.push(this)
  }

  close() {
    this.closed = true
  }

  emit(data: unknown) {
    this.onmessage?.({ data: typeof data === 'string' ? data : JSON.stringify(data) })
  }

  static install() {
    FakeSocket.instances = []
    vi.stubGlobal('WebSocket', FakeSocket)
  }

  static last(): FakeSocket {
    return FakeSocket.instances[FakeSocket.instances.length - 1]
  }
}
