import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'

export const useModelStore = defineStore('model', () => {
  const models = ref([])
  const currentModel = ref(null)
  const spatialTree = ref([])
  const elements = ref([])
  const selectedElementIds = ref(new Set())
  const highlightedElementId = ref(null)
  const loading = ref(false)
  const error = ref(null)

  const selectedElements = computed(() => {
    return elements.value.filter(e => selectedElementIds.value.has(e.id))
  })

  const elementsByCategory = computed(() => {
    const map = {}
    for (const e of elements.value) {
      if (!map[e.category]) map[e.category] = []
      map[e.category].push(e)
    }
    return map
  })

  const elementsByFloor = computed(() => {
    const map = {}
    for (const e of elements.value) {
      const floor = e.floorName || 'Unknown'
      if (!map[floor]) map[floor] = []
      map[floor].push(e)
    }
    return map
  })

  async function fetchModels() {
    loading.value = true
    try {
      const res = await api.get('/models')
      models.value = res.data
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function uploadModel(file, onProgress) {
    loading.value = true
    error.value = null
    try {
      const formData = new FormData()
      formData.append('file', file)

      const res = await api.post('/models', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (progressEvent) => {
          if (onProgress && progressEvent.total) {
            onProgress(Math.round((progressEvent.loaded * 100) / progressEvent.total))
          }
        }
      })
      models.value.unshift(res.data)
      return res.data
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchModel(modelId) {
    loading.value = true
    try {
      const res = await api.get(`/models/${modelId}`)
      currentModel.value = res.data
    } catch (err) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  }

  async function deleteModel(modelId) {
    try {
      await api.delete(`/models/${modelId}`)
      models.value = models.value.filter(m => m.id !== modelId)
    } catch (err) {
      error.value = err.message
    }
  }

  async function fetchSpatialTree(modelId) {
    try {
      const res = await api.get(`/models/${modelId}/tree`)
      spatialTree.value = res.data
    } catch (err) {
      error.value = err.message
    }
  }

  async function fetchElements(modelId, category) {
    try {
      const url = category
        ? `/models/${modelId}/elements?category=${category}`
        : `/models/${modelId}/elements`
      const res = await api.get(url)
      elements.value = res.data
    } catch (err) {
      error.value = err.message
    }
  }

  async function fetchElement(elementId) {
    try {
      const res = await api.get(`/models/${currentModel.value?.id}/elements/${elementId}`)
      return res.data
    } catch (err) {
      error.value = err.message
      return null
    }
  }

  async function fetchMeshChunks(modelId, lod, nodeIds) {
    try {
      const params = nodeIds.length > 0 ? { nodes: nodeIds.join(',') } : {}
      const res = await api.get(`/models/${modelId}/meshes/${lod}`, { params })
      return res.data
    } catch (err) {
      error.value = err.message
      return []
    }
  }

  function selectElement(elementId, additive = false) {
    if (!additive) {
      selectedElementIds.value = new Set([elementId])
    } else {
      const newSet = new Set(selectedElementIds.value)
      if (newSet.has(elementId)) {
        newSet.delete(elementId)
      } else {
        newSet.add(elementId)
      }
      selectedElementIds.value = newSet
    }
    highlightedElementId.value = elementId
  }

  function selectElements(elementIds) {
    selectedElementIds.value = new Set(elementIds)
  }

  function clearSelection() {
    selectedElementIds.value = new Set()
    highlightedElementId.value = null
  }

  function addToSelection(elementId) {
    const newSet = new Set(selectedElementIds.value)
    newSet.add(elementId)
    selectedElementIds.value = newSet
  }

  function removeFromSelection(elementId) {
    const newSet = new Set(selectedElementIds.value)
    newSet.delete(elementId)
    selectedElementIds.value = newSet
  }

  return {
    models, currentModel, spatialTree, elements,
    selectedElementIds, highlightedElementId, loading, error,
    selectedElements, elementsByCategory, elementsByFloor,
    fetchModels, uploadModel, fetchModel, deleteModel,
    fetchSpatialTree, fetchElements, fetchElement, fetchMeshChunks,
    selectElement, selectElements, clearSelection, addToSelection, removeFromSelection
  }
})
