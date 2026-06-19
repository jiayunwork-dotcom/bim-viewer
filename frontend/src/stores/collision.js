import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../utils/api'

export const useCollisionStore = defineStore('collision', () => {
  const groupA = ref([])
  const groupB = ref([])
  const threshold = ref(50)
  const results = ref([])
  const currentTaskId = ref(null)
  const detecting = ref(false)
  const loading = ref(false)

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

  async function detectCollisions(modelId) {
    detecting.value = true
    results.value = []
    try {
      const res = await api.post('/collision/detect', {
        modelId,
        groupA: groupA.value,
        groupB: groupB.value,
        threshold: threshold.value
      })
      currentTaskId.value = res.data.taskId
      results.value = res.data.results || []
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
    } catch (err) {
      console.error('Failed to fetch results:', err)
    } finally {
      loading.value = false
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
  }

  return {
    groupA, groupB, threshold, results, currentTaskId, detecting, loading,
    addToGroupA, removeFromGroupA, addToGroupB, removeFromGroupB,
    addCategoryToGroup, detectCollisions, fetchResults, exportCSV,
    clearGroups, clearResults
  }
})
