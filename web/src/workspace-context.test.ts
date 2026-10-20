import { afterEach, expect, it, vi } from 'vitest'
import { normalizeWorkspaceContext, readWorkspaceContext, saveWorkspaceContext } from './workspace-context'
afterEach(() => vi.unstubAllGlobals())
it('isolates users, restores valid choices and defaults invalid values to all', () => {
 const values = new Map<string,string>()
 vi.stubGlobal('localStorage', { getItem: (key: string) => values.get(key), setItem: (key: string,value: string) => values.set(key,value) })
 expect(readWorkspaceContext(1)).toBe('all')
 expect(saveWorkspaceContext(1,'work')).toBe(true)
 expect(readWorkspaceContext(1)).toBe('work')
 expect(readWorkspaceContext(2)).toBe('all')
 expect(normalizeWorkspaceContext('unclassified')).toBe('all')
})
it('keeps storage exceptions out of the interaction', () => {
 vi.stubGlobal('localStorage', { getItem: () => { throw Error('blocked') }, setItem: () => { throw Error('blocked') } })
 expect(readWorkspaceContext(1)).toBe('all')
 expect(saveWorkspaceContext(1,'life')).toBe(false)
})
