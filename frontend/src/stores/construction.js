import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'

export const useConstructionStore = defineStore('construction', () => {
  const plans = ref([])
  const currentPlan = ref(null)
  const loading = ref(false)
  const playing = ref(false)
  const playSpeed = ref(1)
  const currentDate = ref(null)
  const playProgress = ref(0)
  const playbackActive = ref(false)
  const currentModelId = ref(null)
  const criticalPath = ref(null)
  const hoveredPhaseId = ref(null)

  const PHASE_COLORS = [
    '#409EFF', '#67C23A', '#E6A23C', '#F56C6C',
    '#909399', '#00BCD4', '#9C27B0', '#FF9800',
    '#795548', '#607D8B', '#3F51B5', '#009688'
  ]

  const SPEED_OPTIONS = [1, 2, 4, 8]

  const allPhases = computed(() => {
    if (!currentPlan.value) return []
    return currentPlan.value.phases || []
  })

  const elementPhaseMap = computed(() => {
    const map = new Map()
    if (!currentPlan.value) return map
    for (const phase of currentPlan.value.phases || []) {
      for (const eid of phase.elementIds || []) {
        map.set(eid, phase)
      }
    }
    return map
  })

  function areAllPredecessorsComplete(phase, date) {
    if (!phase || !phase.predecessorIds || phase.predecessorIds.length === 0) {
      return true
    }
    for (const predId of phase.predecessorIds) {
      const predPhase = allPhases.value.find(p => p.id === predId)
      if (!predPhase) continue
      const predOpacity = getPhaseOpacityRaw(predPhase, date)
      if (predOpacity < 1) {
        return false
      }
    }
    return true
  }

  function getPhaseOpacityRaw(phase, date) {
    if (!date || !phase) return 0
    const current = new Date(date).getTime()
    const start = new Date(phase.startDate).getTime()
    const end = new Date(phase.endDate).getTime()
    if (current < start) return 0
    if (current >= end) return 1
    return (current - start) / (end - start)
  }

  function getPhaseOpacity(phase, date) {
    if (!date || !phase) return 0
    if (!areAllPredecessorsComplete(phase, date)) {
      return 0
    }
    return getPhaseOpacityRaw(phase, date)
  }

  function getElementOpacity(elementId, date) {
    if (!date) return 0
    const phase = elementPhaseMap.value.get(elementId)
    if (!phase) return 0
    return getPhaseOpacity(phase, date)
  }

  async function fetchPlans(modelId) {
    loading.value = true
    currentModelId.value = modelId
    try {
      const res = await api.get('/construction/plans', { params: { modelId } })
      plans.value = res.data
    } catch (err) {
      console.error('Failed to fetch construction plans:', err)
    } finally {
      loading.value = false
    }
  }

  async function fetchPlan(planId) {
    loading.value = true
    try {
      const res = await api.get(`/construction/plans/${planId}`)
      currentPlan.value = res.data
      await fetchCriticalPath(planId)
    } catch (err) {
      console.error('Failed to fetch construction plan:', err)
    } finally {
      loading.value = false
    }
  }

  async function createPlan(data) {
    try {
      const res = await api.post('/construction/plans', data)
      plans.value.unshift(res.data)
      return res.data
    } catch (err) {
      console.error('Failed to create construction plan:', err)
      throw err
    }
  }

  async function updatePlan(planId, data) {
    try {
      const res = await api.put(`/construction/plans/${planId}`, data)
      const idx = plans.value.findIndex(p => p.id === planId)
      if (idx > -1) plans.value[idx] = res.data
      if (currentPlan.value?.id === planId) currentPlan.value = res.data
      return res.data
    } catch (err) {
      console.error('Failed to update construction plan:', err)
      throw err
    }
  }

  async function deletePlan(planId) {
    try {
      await api.delete(`/construction/plans/${planId}`)
      plans.value = plans.value.filter(p => p.id !== planId)
      if (currentPlan.value?.id === planId) {
        currentPlan.value = null
        stopPlayback()
      }
    } catch (err) {
      console.error('Failed to delete construction plan:', err)
      throw err
    }
  }

  async function createPhase(planId, data) {
    try {
      const res = await api.post(`/construction/plans/${planId}/phases`, data)
      if (currentPlan.value?.id === planId) {
        await fetchPlan(planId)
      }
      return res.data
    } catch (err) {
      console.error('Failed to create phase:', err)
      throw err
    }
  }

  async function updatePhase(planId, phaseId, data) {
    try {
      const res = await api.put(`/construction/plans/${planId}/phases/${phaseId}`, data)
      if (currentPlan.value?.id === planId) {
        await fetchPlan(planId)
      }
      return res.data
    } catch (err) {
      console.error('Failed to update phase:', err)
      throw err
    }
  }

  async function deletePhase(planId, phaseId) {
    try {
      await api.delete(`/construction/plans/${planId}/phases/${phaseId}`)
      if (currentPlan.value?.id === planId) {
        await fetchPlan(planId)
        await fetchCriticalPath(planId)
      }
    } catch (err) {
      console.error('Failed to delete phase:', err)
      throw err
    }
  }

  async function fetchCriticalPath(planId) {
    try {
      const res = await api.get(`/construction/plans/${planId}/critical-path`)
      criticalPath.value = res.data
      return res.data
    } catch (err) {
      console.error('Failed to fetch critical path:', err)
      criticalPath.value = null
      throw err
    }
  }

  async function updatePhasePredecessors(planId, phaseId, predecessorIds) {
    try {
      const res = await api.put(`/construction/plans/${planId}/phases/${phaseId}`, {
        predecessorIds
      })
      if (currentPlan.value?.id === planId) {
        await fetchPlan(planId)
        await fetchCriticalPath(planId)
      }
      return res.data
    } catch (err) {
      console.error('Failed to update phase predecessors:', err)
      throw err
    }
  }

  function setHoveredPhaseId(phaseId) {
    hoveredPhaseId.value = phaseId
  }

  function isPhaseOnCriticalPath(phaseId) {
    if (!criticalPath.value?.phaseIds) return false
    return criticalPath.value.phaseIds.includes(phaseId)
  }

  function hasDirectDependency(phaseIdA, phaseIdB) {
    const phaseA = allPhases.value.find(p => p.id === phaseIdA)
    const phaseB = allPhases.value.find(p => p.id === phaseIdB)
    if (!phaseA || !phaseB) return false
    return (phaseA.predecessorIds || []).includes(phaseIdB) ||
           (phaseB.predecessorIds || []).includes(phaseIdA)
  }

  function getDependencyArrows() {
    const arrows = []
    const phases = allPhases.value
    for (const phase of phases) {
      for (const predId of phase.predecessorIds || []) {
        const predPhase = phases.find(p => p.id === predId)
        if (predPhase) {
          arrows.push({
            from: predPhase.id,
            to: phase.id,
            fromPhase: predPhase,
            toPhase: phase
          })
        }
      }
    }
    return arrows
  }

  function setCurrentPlan(plan) {
    currentPlan.value = plan
    if (plan) {
      currentDate.value = plan.startDate
      playProgress.value = 0
      fetchCriticalPath(plan.id)
    } else {
      currentDate.value = null
      playProgress.value = 0
      criticalPath.value = null
    }
  }

  function startPlayback() {
    if (!currentPlan.value) return
    playbackActive.value = true
    playing.value = true
  }

  function pausePlayback() {
    playing.value = false
  }

  function stopPlayback() {
    playing.value = false
    playbackActive.value = false
    currentDate.value = null
    playProgress.value = 0
  }

  function setPlaySpeed(speed) {
    playSpeed.value = speed
  }

  function seekToProgress(progress) {
    if (!currentPlan.value) return
    playProgress.value = Math.max(0, Math.min(1, progress))
    const start = new Date(currentPlan.value.startDate).getTime()
    const end = new Date(currentPlan.value.endDate).getTime()
    const current = start + playProgress.value * (end - start)
    currentDate.value = new Date(current).toISOString().split('T')[0]
  }

  function seekToDate(dateStr) {
    if (!currentPlan.value) return
    const start = new Date(currentPlan.value.startDate).getTime()
    const end = new Date(currentPlan.value.endDate).getTime()
    const current = new Date(dateStr).getTime()
    playProgress.value = Math.max(0, Math.min(1, (current - start) / (end - start)))
    currentDate.value = dateStr
  }

  function advancePlayback(deltaDays) {
    if (!currentPlan.value || !currentDate.value) return
    const current = new Date(currentDate.value)
    current.setDate(current.getDate() + deltaDays)
    const start = new Date(currentPlan.value.startDate)
    const end = new Date(currentPlan.value.endDate)
    if (current > end) {
      currentDate.value = currentPlan.value.endDate
      playProgress.value = 1
      playing.value = false
      return
    }
    if (current < start) {
      currentDate.value = currentPlan.value.startDate
      playProgress.value = 0
      return
    }
    currentDate.value = current.toISOString().split('T')[0]
    const startTime = start.getTime()
    const endTime = end.getTime()
    playProgress.value = (current.getTime() - startTime) / (endTime - startTime)
  }

  return {
    plans, currentPlan, loading, playing, playSpeed, currentDate,
    playProgress, playbackActive, currentModelId, criticalPath, hoveredPhaseId,
    PHASE_COLORS, SPEED_OPTIONS, allPhases, elementPhaseMap,
    getPhaseOpacity, getElementOpacity, areAllPredecessorsComplete,
    fetchPlans, fetchPlan, createPlan, updatePlan, deletePlan,
    createPhase, updatePhase, deletePhase,
    fetchCriticalPath, updatePhasePredecessors,
    setHoveredPhaseId, isPhaseOnCriticalPath, hasDirectDependency, getDependencyArrows,
    setCurrentPlan, startPlayback, pausePlayback, stopPlayback,
    setPlaySpeed, seekToProgress, seekToDate, advancePlayback
  }
})
