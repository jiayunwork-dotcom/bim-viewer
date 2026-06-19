import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'

export const useCollisionStore = defineStore('collision', () => {
  const groupA = ref([])
  const groupB = ref([])
  const threshold = ref(50)
  const results = ref([])
  const currentTaskId = ref(null)
  const detecting = ref(false)
  const loading = ref(false)
  const stats = ref({ total: 0, pending: 0, confirmed: 0, false_positive: 0, resolved: 0 })
  const statusFilter = ref('all')
  const severityFilter = ref('all')
  const selectedIds = ref([])
  const historyMap = ref({})
  const historyLoading = ref(false)
  const currentModelId = ref(null)

  const COLLISION_STATUS = {
    PENDING: 'pending',
    CONFIRMED: 'confirmed',
    FALSE_POSITIVE: 'false_positive',
    RESOLVED: 'resolved'
  }

  const STATUS_LABELS = {
    [COLLISION_STATUS.PENDING]: '待处理',
    [COLLISION_STATUS.CONFIRMED]: '已确认',
    [COLLISION_STATUS.FALSE_POSITIVE]: '误报',
    [COLLISION_STATUS.RESOLVED]: '已解决'
  }

  const STATUS_TYPES = {
    [COLLISION_STATUS.PENDING]: 'info',
    [COLLISION_STATUS.CONFIRMED]: 'danger',
    [COLLISION_STATUS.FALSE_POSITIVE]: 'success',
    [COLLISION_STATUS.RESOLVED]: 'primary'
  }

  const filteredResults = computed(() => {
    let filtered = results.value
    if (statusFilter.value !== 'all') {
      filtered = filtered.filter(r => r.status === statusFilter.value)
    }
    if (severityFilter.value !== 'all') {
      filtered = filtered.filter(r => r.collisionType === severityFilter.value)
    }
    return filtered
  })

  const allSelected = computed(() => {
    return filteredResults.value.length > 0 && filteredResults.value.every(r => selectedIds.value.includes(r.id))
  })

  const hasSelection = computed(() => selectedIds.value.length > 0)

  function recalcStats() {
    const s = { total: 0, pending: 0, confirmed: 0, false_positive: 0, resolved: 0 }
    for (const r of results.value) {
      s.total++
      if (r.status === 'pending') s.pending++
      else if (r.status === 'confirmed') s.confirmed++
      else if (r.status === 'false_positive') s.false_positive++
      else if (r.status === 'resolved') s.resolved++
    }
    stats.value = s
  }

  function addToGroupA(elementId) {
    if (!groupA.value.includes(elementId)) {
      groupA.value.push(elementId)
    }
  }

  function removeFromGroupA(elementId) {
    groupA.value = groupA.value.filter(id => id !== elementId)
  }

  function addToGroupB(elementId) {
    if (!groupB.value.includes(elementId)) {
      groupB.value.push(elementId)
    }
  }

  function removeFromGroupB(elementId) {
    groupB.value = groupB.value.filter(id => id !== elementId)
  }

  function addCategoryToGroup(modelStore, category, group) {
    const elems = modelStore.elements.filter(e => e.category === category)
    const ids = elems.map(e => e.id)
    if (group === 'A') {
      groupA.value = [...new Set([...groupA.value, ...ids])]
    } else {
      groupB.value = [...new Set([...groupB.value, ...ids])]
    }
  }

  function toggleSelection(id) {
    const index = selectedIds.value.indexOf(id)
    if (index > -1) {
      selectedIds.value.splice(index, 1)
    } else {
      selectedIds.value.push(id)
    }
  }

  function toggleAllSelection() {
    if (allSelected.value) {
      selectedIds.value = []
    } else {
      selectedIds.value = filteredResults.value.map(r => r.id)
    }
  }

  function clearSelection() {
    selectedIds.value = []
  }

  async function detectCollisions(modelId) {
    detecting.value = true
    results.value = []
    selectedIds.value = []
    currentModelId.value = modelId
    try {
      const res = await api.post('/collision/detect', {
        modelId,
        groupA: groupA.value,
        groupB: groupB.value,
        threshold: threshold.value
      })
      currentTaskId.value = res.data.taskId
      results.value = res.data.results || []
      recalcStats()
      return res.data
    } catch (err) {
      console.error('Collision detection failed:', err)
      throw err
    } finally {
      detecting.value = false
    }
  }

  async function fetchResults(taskId) {
    loading.value = true
    try {
      const res = await api.get(`/collision/results/${taskId}`)
      results.value = res.data
      recalcStats()
    } catch (err) {
      console.error('Failed to fetch results:', err)
    } finally {
      loading.value = false
    }
  }

  async function fetchResultsByModel(modelId) {
    loading.value = true
    currentModelId.value = modelId
    try {
      const res = await api.get(`/collision/model/${modelId}/results`)
      results.value = res.data
      if (res.data.length > 0) {
        currentTaskId.value = res.data[0].taskId
      }
      recalcStats()
    } catch (err) {
      console.error('Failed to fetch results by model:', err)
    } finally {
      loading.value = false
    }
  }

  async function fetchStats(taskId) {
    try {
      const res = await api.get(`/collision/stats/${taskId}`)
      stats.value = res.data
    } catch (err) {
      console.error('Failed to fetch stats:', err)
    }
  }

  async function fetchStatsByModel(modelId) {
    try {
      const res = await api.get(`/collision/model/${modelId}/stats`)
      stats.value = res.data
    } catch (err) {
      console.error('Failed to fetch stats by model:', err)
    }
  }

  async function fetchHistory(resultId) {
    historyLoading.value = true
    try {
      const res = await api.get(`/collision/history/${resultId}`)
      historyMap.value[resultId] = res.data
      return res.data
    } catch (err) {
      console.error('Failed to fetch history:', err)
      throw err
    } finally {
      historyLoading.value = false
    }
  }

  function applyStatusToResult(resultId, newStatus) {
    const idx = results.value.findIndex(r => r.id === resultId)
    if (idx > -1) {
      const updated = { ...results.value[idx], status: newStatus, updatedAt: new Date().toISOString() }
      results.value.splice(idx, 1, updated)
    }
  }

  async function updateStatus(resultId, newStatus, remark, operator = 'user') {
    try {
      await api.put(`/collision/results/${resultId}/status`, {
        newStatus,
        remark,
        operator
      })
      applyStatusToResult(resultId, newStatus)
      recalcStats()
    } catch (err) {
      console.error('Failed to update status:', err)
      throw err
    }
  }

  async function batchUpdateStatus(newStatus, remark, operator = 'user') {
    const targetIds = [...selectedIds.value]
    try {
      await api.put('/collision/results/batch/status', {
        resultIds: targetIds,
        newStatus,
        remark,
        operator
      })
      for (const id of targetIds) {
        applyStatusToResult(id, newStatus)
      }
      selectedIds.value = []
      recalcStats()
    } catch (err) {
      console.error('Failed to batch update status:', err)
      throw err
    }
  }

  async function exportCSV(taskId) {
    try {
      const res = await api.get(`/collision/export/${taskId}`, {
        responseType: 'blob'
      })
      const blob = new Blob([res.data], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `collision_report_${taskId}.csv`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
    } catch (err) {
      console.error('Failed to export CSV:', err)
      throw err
    }
  }

  function clearGroups() {
    groupA.value = []
    groupB.value = []
  }

  function clearResults() {
    results.value = []
    currentTaskId.value = null
    currentModelId.value = null
    stats.value = { total: 0, pending: 0, confirmed: 0, false_positive: 0, resolved: 0 }
    selectedIds.value = []
    historyMap.value = {}
  }

  function setStatusFilter(filter) {
    statusFilter.value = filter
  }

  function setSeverityFilter(filter) {
    severityFilter.value = filter
  }

  return {
    groupA, groupB, threshold, results, currentTaskId, detecting, loading,
    stats, statusFilter, severityFilter, selectedIds, historyMap, historyLoading,
    currentModelId,
    COLLISION_STATUS, STATUS_LABELS, STATUS_TYPES,
    filteredResults, allSelected, hasSelection,
    addToGroupA, removeFromGroupA, addToGroupB, removeFromGroupB,
    addCategoryToGroup, toggleSelection, toggleAllSelection, clearSelection,
    detectCollisions, fetchResults, fetchResultsByModel,
    fetchStats, fetchStatsByModel, fetchHistory,
    updateStatus, batchUpdateStatus, exportCSV,
    clearGroups, clearResults, setStatusFilter, setSeverityFilter
  }
})
