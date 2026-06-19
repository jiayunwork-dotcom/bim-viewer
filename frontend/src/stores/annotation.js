import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../utils/api'

export const useAnnotationStore = defineStore('annotation', () => {
  const annotations = ref([])
  const currentAnnotation = ref(null)
  const loading = ref(false)
  const creating = ref(false)
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(20)
  const totalPages = ref(0)
  const priorityFilter = ref('')
  const statusFilter = ref('')
  const sortBy = ref('createdAt')
  const currentModelId = ref(null)

  const ws = ref(null)
  const wsConnected = ref(false)
  const wsReconnectTimer = ref(null)
  const lastMessageTime = ref(null)

  const PRIORITY_COLORS = {
    urgent: '#ff4444',
    normal: '#409eff',
    low: '#909399'
  }

  const PRIORITY_LABELS = {
    urgent: '紧急',
    normal: '普通',
    low: '低'
  }

  const STATUS_LABELS = {
    open: '打开',
    in_progress: '处理中',
    closed: '已关闭'
  }

  const STATUS_TYPES = {
    open: 'danger',
    in_progress: 'warning',
    closed: 'success'
  }

  const filteredAnnotations = computed(() => {
    let filtered = annotations.value
    if (priorityFilter.value) {
      filtered = filtered.filter(a => a.priority === priorityFilter.value)
    }
    if (statusFilter.value) {
      filtered = filtered.filter(a => a.status === statusFilter.value)
    }
    return filtered
  })

  function getPinColor(annotation) {
    if (annotation.status === 'closed') return PRIORITY_COLORS[annotation.priority] + '66'
    return PRIORITY_COLORS[annotation.priority]
  }

  function getPinOpacity(annotation) {
    return annotation.status === 'closed' ? 0.4 : 1.0
  }

  async function fetchAnnotations(modelId, resetPage = false) {
    if (resetPage) page.value = 1
    loading.value = true
    currentModelId.value = modelId
    try {
      const params = new URLSearchParams({
        modelId,
        page: page.value,
        pageSize: pageSize.value,
        sortBy: sortBy.value
      })
      if (priorityFilter.value) params.set('priority', priorityFilter.value)
      if (statusFilter.value) params.set('status', statusFilter.value)

      const res = await api.get(`/annotations?${params}`)
      annotations.value = res.data.items || []
      total.value = res.data.total || 0
      page.value = res.data.page || 1
      totalPages.value = res.data.totalPages || 0
    } catch (err) {
      console.error('Failed to fetch annotations:', err)
    } finally {
      loading.value = false
    }
  }

  async function fetchAnnotation(id) {
    loading.value = true
    try {
      const res = await api.get(`/annotations/${id}`)
      currentAnnotation.value = res.data
      return res.data
    } catch (err) {
      console.error('Failed to fetch annotation:', err)
    } finally {
      loading.value = false
    }
  }

  async function createAnnotation(formData) {
    creating.value = true
    try {
      const res = await api.post('/annotations', formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
      const newAnnotation = res.data
      annotations.value.unshift(newAnnotation)
      total.value++
      return newAnnotation
    } catch (err) {
      console.error('Failed to create annotation:', err)
      throw err
    } finally {
      creating.value = false
    }
  }

  async function updateAnnotation(id, data) {
    try {
      const res = await api.put(`/annotations/${id}`, data)
      const updated = res.data
      const idx = annotations.value.findIndex(a => a.id === id)
      if (idx > -1) {
        annotations.value.splice(idx, 1, { ...annotations.value[idx], ...updated })
      }
      if (currentAnnotation.value?.id === id) {
        currentAnnotation.value = { ...currentAnnotation.value, ...updated }
      }
      return updated
    } catch (err) {
      console.error('Failed to update annotation:', err)
      throw err
    }
  }

  async function deleteAnnotation(id) {
    try {
      await api.delete(`/annotations/${id}`)
      annotations.value = annotations.value.filter(a => a.id !== id)
      total.value--
      if (currentAnnotation.value?.id === id) {
        currentAnnotation.value = null
      }
    } catch (err) {
      console.error('Failed to delete annotation:', err)
      throw err
    }
  }

  async function addComment(annotationId, formData) {
    try {
      const res = await api.post(`/annotations/${annotationId}/comments`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
      const newComment = res.data
      if (currentAnnotation.value?.id === annotationId) {
        currentAnnotation.value.comments = currentAnnotation.value.comments || []
        currentAnnotation.value.comments.push(newComment)
      }
      const idx = annotations.value.findIndex(a => a.id === annotationId)
      if (idx > -1) {
        annotations.value[idx].comments = annotations.value[idx].comments || []
        annotations.value[idx].comments.push(newComment)
      }
      return newComment
    } catch (err) {
      console.error('Failed to add comment:', err)
      throw err
    }
  }

  function connectWebSocket(modelId) {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      if (currentModelId.value === modelId) return
      disconnectWebSocket()
    }

    currentModelId.value = modelId
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const wsUrl = `${protocol}//${host}/ws/annotations?modelId=${modelId}`

    ws.value = new WebSocket(wsUrl)

    ws.value.onopen = () => {
      wsConnected.value = true
      if (wsReconnectTimer.value) {
        clearTimeout(wsReconnectTimer.value)
        wsReconnectTimer.value = null
      }
      if (lastMessageTime.value) {
        ws.value.send(JSON.stringify({
          type: 'sync_request',
          since: lastMessageTime.value
        }))
      }
    }

    ws.value.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        lastMessageTime.value = msg.timestamp || new Date().toISOString()
        handleWSMessage(msg)
      } catch (e) {
        console.error('Failed to parse WS message:', e)
      }
    }

    ws.value.onclose = () => {
      wsConnected.value = false
      scheduleReconnect(modelId)
    }

    ws.value.onerror = () => {
      wsConnected.value = false
    }
  }

  function disconnectWebSocket() {
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
    wsConnected.value = false
    if (wsReconnectTimer.value) {
      clearTimeout(wsReconnectTimer.value)
      wsReconnectTimer.value = null
    }
  }

  function scheduleReconnect(modelId) {
    if (wsReconnectTimer.value) return
    wsReconnectTimer.value = setTimeout(() => {
      wsReconnectTimer.value = null
      connectWebSocket(modelId)
    }, 3000)
  }

  function handleWSMessage(msg) {
    switch (msg.type) {
      case 'annotation_created':
        if (msg.payload && !annotations.value.find(a => a.id === msg.payload.id)) {
          annotations.value.unshift(msg.payload)
          total.value++
        }
        break
      case 'annotation_updated':
        if (msg.payload) {
          const idx = annotations.value.findIndex(a => a.id === msg.payload.id)
          if (idx > -1) {
            annotations.value.splice(idx, 1, { ...annotations.value[idx], ...msg.payload })
          }
          if (currentAnnotation.value?.id === msg.payload.id) {
            currentAnnotation.value = { ...currentAnnotation.value, ...msg.payload }
          }
        }
        break
      case 'annotation_deleted':
        if (msg.payload?.id) {
          annotations.value = annotations.value.filter(a => a.id !== msg.payload.id)
          total.value--
          if (currentAnnotation.value?.id === msg.payload.id) {
            currentAnnotation.value = null
          }
        }
        break
      case 'comment_added':
        if (msg.payload) {
          const annIdx = annotations.value.findIndex(a => a.id === msg.payload.annotationId)
          if (annIdx > -1) {
            annotations.value[annIdx].comments = annotations.value[annIdx].comments || []
            annotations.value[annIdx].comments.push(msg.payload)
          }
          if (currentAnnotation.value?.id === msg.payload.annotationId) {
            currentAnnotation.value.comments = currentAnnotation.value.comments || []
            currentAnnotation.value.comments.push(msg.payload)
          }
        }
        break
      case 'sync_response':
        if (Array.isArray(msg.payload)) {
          for (const ann of msg.payload) {
            const existingIdx = annotations.value.findIndex(a => a.id === ann.id)
            if (existingIdx > -1) {
              annotations.value.splice(existingIdx, 1, { ...annotations.value[existingIdx], ...ann })
            } else {
              annotations.value.unshift(ann)
              total.value++
            }
          }
        }
        break
    }
  }

  function setCurrentAnnotation(annotation) {
    currentAnnotation.value = annotation
  }

  function setPriorityFilter(val) {
    priorityFilter.value = val
  }

  function setStatusFilter(val) {
    statusFilter.value = val
  }

  function setSortBy(val) {
    sortBy.value = val
  }

  function setPage(val) {
    page.value = val
  }

  function clearAll() {
    annotations.value = []
    currentAnnotation.value = null
    total.value = 0
    page.value = 1
    currentModelId.value = null
    disconnectWebSocket()
  }

  return {
    annotations, currentAnnotation, loading, creating,
    total, page, pageSize, totalPages,
    priorityFilter, statusFilter, sortBy, currentModelId,
    ws, wsConnected, lastMessageTime,
    PRIORITY_COLORS, PRIORITY_LABELS, STATUS_LABELS, STATUS_TYPES,
    filteredAnnotations,
    getPinColor, getPinOpacity,
    fetchAnnotations, fetchAnnotation,
    createAnnotation, updateAnnotation, deleteAnnotation,
    addComment,
    connectWebSocket, disconnectWebSocket,
    setCurrentAnnotation,
    setPriorityFilter, setStatusFilter, setSortBy, setPage,
    clearAll
  }
})
