import { afterEach, expect, it, vi } from 'vitest'
import { notifyTaskChange, subscribeTaskChanges } from './task-changes'
afterEach(() => vi.unstubAllGlobals())
it('notifies local and remote tabs without sharing task data and unsubscribes cleanly', () => {
 const target = new EventTarget(), saved = vi.fn()
 vi.stubGlobal('window', target); vi.stubGlobal('localStorage', { setItem: saved })
 const refresh = vi.fn(), stop = subscribeTaskChanges(refresh)
 notifyTaskChange(); expect(refresh).toHaveBeenCalledTimes(1)
 expect(saved.mock.calls[0]?.[0]).toBe('learnos:tasks-changed')
 const event = new Event('storage'); Object.defineProperty(event, 'key', { value: 'learnos:tasks-changed' })
 target.dispatchEvent(event); expect(refresh).toHaveBeenCalledTimes(2)
 stop(); notifyTaskChange(); expect(refresh).toHaveBeenCalledTimes(2)
})
it('still refreshes when storage is unavailable', () => {
 vi.stubGlobal('window', new EventTarget()); vi.stubGlobal('localStorage', { setItem: () => { throw Error('blocked') } })
 const refresh = vi.fn(), stop = subscribeTaskChanges(refresh)
 expect(() => notifyTaskChange()).not.toThrow(); expect(refresh).toHaveBeenCalledTimes(1); stop()
})
