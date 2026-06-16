/* HUB-GROUPS-V1: localStorage key "agenthub:hubGroups:v1" — JSON array of HubGroupDef */

export interface HubGroupDef {
  id: string           // random uuid — stable across restarts
  name: string         // user-chosen display name
  memberKeys: string[] // "${session.name}:::${session.workDir}" strings
}

const STORAGE_KEY = 'agenthub:hubGroups:v1'

/* GROUP-04: membership key = "${session.name}:::${session.workDir}" — survives session-id churn */
export function memberKey(name: string, workDir: string): string {
  return `${name}:::${workDir || '__nodir__'}`
}

export function loadGroups(): HubGroupDef[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? (JSON.parse(raw) as HubGroupDef[]) : []
  } catch {
    return []
  }
}

export function saveGroups(groups: HubGroupDef[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(groups))
}

export function createGroup(groups: HubGroupDef[], name: string): HubGroupDef[] {
  const def: HubGroupDef = { id: crypto.randomUUID(), name, memberKeys: [] }
  const updated = [...groups, def]
  saveGroups(updated)
  return updated
}

export function assignToGroup(
  groups: HubGroupDef[],
  groupId: string,
  key: string,
): HubGroupDef[] {
  const updated = groups.map((g) => ({
    ...g,
    memberKeys:
      g.id === groupId
        ? [...g.memberKeys.filter((k) => k !== key), key]
        : g.memberKeys.filter((k) => k !== key),
  }))
  saveGroups(updated)
  return updated
}

export function removeFromGroup(groups: HubGroupDef[], key: string): HubGroupDef[] {
  const updated = groups.map((g) => ({
    ...g,
    memberKeys: g.memberKeys.filter((k) => k !== key),
  }))
  saveGroups(updated)
  return updated
}

export function deleteGroup(groups: HubGroupDef[], groupId: string): HubGroupDef[] {
  const updated = groups.filter((g) => g.id !== groupId)
  saveGroups(updated)
  return updated
}
