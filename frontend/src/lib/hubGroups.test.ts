import { describe, it, expect, beforeEach } from 'vitest'
import {
  memberKey,
  loadGroups,
  saveGroups,
  createGroup,
  assignToGroup,
  removeFromGroup,
  type HubGroupDef,
} from './hubGroups'

// Each test suite starts with a clean localStorage slate.
beforeEach(() => {
  localStorage.clear()
})

describe('memberKey', () => {
  it('derives key from name and workDir', () => {
    expect(memberKey('claude', '/home/ken/proj')).toBe('claude:::/home/ken/proj')
  })

  it('maps empty workDir to __nodir__', () => {
    expect(memberKey('claude', '')).toBe('claude:::__nodir__')
  })
})

describe('loadGroups / saveGroups', () => {
  it('returns [] when localStorage key is absent', () => {
    expect(loadGroups()).toEqual([])
  })

  it('returns [] on malformed JSON (resilience)', () => {
    localStorage.setItem('agenthub:hubGroups:v1', '{not-valid-json')
    expect(loadGroups()).toEqual([])
  })

  it('round-trips a HubGroupDef[] through localStorage key agenthub:hubGroups:v1', () => {
    const groups: HubGroupDef[] = [
      { id: 'g1', name: 'Alpha', memberKeys: ['claude:::__nodir__'] },
    ]
    saveGroups(groups)
    expect(loadGroups()).toEqual(groups)
  })
})

describe('createGroup', () => {
  it('appends a new group with a non-empty id, given name, and empty memberKeys', () => {
    const start: HubGroupDef[] = []
    const updated = createGroup(start, 'My Group')
    expect(updated).toHaveLength(1)
    expect(updated[0].name).toBe('My Group')
    expect(updated[0].id).toBeTruthy()
    expect(updated[0].memberKeys).toEqual([])
  })

  it('persists the new group so loadGroups() returns it (GROUP-03 persistence)', () => {
    createGroup([], 'Persistent Group')
    const reloaded = loadGroups()
    expect(reloaded).toHaveLength(1)
    expect(reloaded[0].name).toBe('Persistent Group')
  })
})

describe('assignToGroup', () => {
  it('adds key to target group and removes it from every other group', () => {
    let groups = createGroup([], 'GroupA')
    groups = createGroup(groups, 'GroupB')
    const [a, b] = groups
    // First put key in GroupA
    groups = assignToGroup(groups, a.id, 'session:::__nodir__')
    expect(groups.find((g) => g.id === a.id)!.memberKeys).toContain('session:::__nodir__')
    // Now move to GroupB — must leave GroupA
    groups = assignToGroup(groups, b.id, 'session:::__nodir__')
    expect(groups.find((g) => g.id === a.id)!.memberKeys).not.toContain('session:::__nodir__')
    expect(groups.find((g) => g.id === b.id)!.memberKeys).toContain('session:::__nodir__')
  })

  it('does not duplicate a key already present in the target', () => {
    let groups = createGroup([], 'GroupA')
    const [a] = groups
    groups = assignToGroup(groups, a.id, 'key:::__nodir__')
    groups = assignToGroup(groups, a.id, 'key:::__nodir__')
    expect(groups.find((g) => g.id === a.id)!.memberKeys).toHaveLength(1)
  })

  it('persists the result (GROUP-03)', () => {
    let groups = createGroup([], 'G')
    const [g] = groups
    groups = assignToGroup(groups, g.id, 'sess:::__nodir__')
    const reloaded = loadGroups()
    expect(reloaded.find((gr) => gr.id === g.id)!.memberKeys).toContain('sess:::__nodir__')
  })
})

describe('removeFromGroup', () => {
  it('strips a key from all groups', () => {
    let groups = createGroup([], 'GroupA')
    const [a] = groups
    groups = assignToGroup(groups, a.id, 'rem:::__nodir__')
    groups = removeFromGroup(groups, 'rem:::__nodir__')
    expect(groups.find((g) => g.id === a.id)!.memberKeys).not.toContain('rem:::__nodir__')
  })
})

describe('GROUP-03 persistence: remount simulation', () => {
  it('fresh loadGroups() after saveGroups() returns the same data', () => {
    const groups: HubGroupDef[] = [
      { id: 'abc', name: 'Alpha', memberKeys: ['s1:::__nodir__'] },
      { id: 'def', name: 'Beta', memberKeys: [] },
    ]
    saveGroups(groups)
    // Simulate remount — call loadGroups() fresh
    const reloaded = loadGroups()
    expect(reloaded).toEqual(groups)
  })
})
