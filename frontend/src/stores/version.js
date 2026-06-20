import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'

export const useVersionStore = defineStore('version', () => {
  const versions = ref([])
  const currentBaseVersion = ref(null)
  const currentCompareVersion = ref(null)
  const diffResult = ref(null)
  const compareMode = ref(false)
  const loading = ref(false)
  const creating = ref(false)
  const filterType = ref('all')
  const selectedElementId = ref(null)
  const elementPropertyCache = ref({})

  const DIFF_TYPES = {
    ADDED: 'added',
    REMOVED: 'removed',
    MODIFIED: 'modified',
    UNCHANGED: 'unchanged'
  }

  const DIFF_COLORS = {
    [DIFF_TYPES.ADDED]: 0x00ff00,
    [DIFF_TYPES.REMOVED]: 0xff0000,
    [DIFF_TYPES.MODIFIED]: 0xffff00,
    [DIFF_TYPES.UNCHANGED]: 0x888888
  }

  const DIFF_COLORS_CSS = {
    [DIFF_TYPES.ADDED]: '#00ff00',
    [DIFF_TYPES.REMOVED]: '#ff0000',
    [DIFF_TYPES.MODIFIED]: '#ffff00',
    [DIFF_TYPES.UNCHANGED]: '#888888'
  }

  const stats = computed(() => {
    if (!diffResult.value?.diff) {
      return {
        added: 0,
        removed: 0,
        modified: 0,
        unchanged: 0,
        total: 0
      }
    }
    const d = diffResult.value.diff
    return {
      added: d.added.length,
      removed: d.removed.length,
      modified: d.modified.length,
      unchanged: d.unchanged.length,
      total: d.added.length + d.removed.length + d.modified.length + d.unchanged.length
    }
  })

  const elementDiffMap = computed(() => {
    const map = {}
    if (!diffResult.value?.diff) return map
    
    for (const id of diffResult.value.diff.added) {
      map[id] = DIFF_TYPES.ADDED
    }
    for (const id of diffResult.value.diff.removed) {
      map[id] = DIFF_TYPES.REMOVED
    }
    for (const id of diffResult.value.diff.modified) {
      map[id] = DIFF_TYPES.MODIFIED
    }
    for (const id of diffResult.value.diff.unchanged) {
      map[id] = DIFF_TYPES.UNCHANGED
    }
    return map
  })

  const filteredElementIds = computed(() => {
    if (!diffResult.value?.diff) return []
    const d = diffResult.value.diff
    
    switch (filterType.value) {
      case DIFF_TYPES.ADDED:
        return d.added
      case DIFF_TYPES.REMOVED:
        return d.removed
      case DIFF_TYPES.MODIFIED:
        return d.modified
      case DIFF_TYPES.UNCHANGED:
        return d.unchanged
      default:
        return [...d.added, ...d.removed, ...d.modified, ...d.unchanged]
    }
  })

  async function fetchVersions(modelId) {
    loading.value = true
    try {
      const res = await api.get(`/models/${modelId}/versions`)
      versions.value = res.data
    } catch (err) {
      console.error('Failed to fetch versions:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createVersion(modelId, description) {
    creating.value = true
    try {
      const res = await api.post(`/models/${modelId}/versions`, { description })
      versions.value.unshift(res.data)
      return res.data
    } catch (err) {
      console.error('Failed to create version:', err)
      throw err
    } finally {
      creating.value = false
    }
  }

  async function deleteVersion(versionId) {
    try {
      await api.delete(`/versions/${versionId}`)
      versions.value = versions.value.filter(v => v.id !== versionId)
    } catch (err) {
      console.error('Failed to delete version:', err)
      throw err
    }
  }

  async function compareVersions(modelId, baseVersionId, compareVersionId) {
    loading.value = true
    try {
      const res = await api.post(`/models/${modelId}/versions/compare`, {
        baseVersionId,
        compareVersionId
      })
      diffResult.value = res.data
      currentBaseVersion.value = res.data.baseVersion
      currentCompareVersion.value = res.data.compareVersion
      compareMode.value = true
      filterType.value = 'all'
      selectedElementId.value = null
      return res.data
    } catch (err) {
      console.error('Failed to compare versions:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function getVersionElement(versionId, elementId) {
    const cacheKey = `${versionId}_${elementId}`
    if (elementPropertyCache.value[cacheKey]) {
      return elementPropertyCache.value[cacheKey]
    }
    try {
      const res = await api.get(`/versions/${versionId}/elements/${elementId}`)
      elementPropertyCache.value[cacheKey] = res.data
      return res.data
    } catch (err) {
      console.error('Failed to get version element:', err)
      return null
    }
  }

  function getElementDiffType(elementId) {
    return elementDiffMap.value[elementId] || null
  }

  function getElementDiffColor(elementId) {
    const type = getElementDiffType(elementId)
    return type ? DIFF_COLORS[type] : null
  }

  function setFilterType(type) {
    filterType.value = type
  }

  function selectElement(elementId) {
    selectedElementId.value = elementId
  }

  function exitCompareMode() {
    compareMode.value = false
    diffResult.value = null
    currentBaseVersion.value = null
    currentCompareVersion.value = null
    filterType.value = 'all'
    selectedElementId.value = null
    elementPropertyCache.value = {}
  }

  function clearStore() {
    exitCompareMode()
    versions.value = []
  }

  return {
    versions,
    currentBaseVersion,
    currentCompareVersion,
    diffResult,
    compareMode,
    loading,
    creating,
    filterType,
    selectedElementId,
    elementPropertyCache,
    DIFF_TYPES,
    DIFF_COLORS,
    DIFF_COLORS_CSS,
    stats,
    elementDiffMap,
    filteredElementIds,
    fetchVersions,
    createVersion,
    deleteVersion,
    compareVersions,
    getVersionElement,
    getElementDiffType,
    getElementDiffColor,
    setFilterType,
    selectElement,
    exitCompareMode,
    clearStore
  }
})
