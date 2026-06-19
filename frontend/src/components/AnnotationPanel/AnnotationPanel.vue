<template>
  <div class="annotation-panel">
    <div v-if="!currentAnnotation" class="annotation-list">
      <div class="panel-header">
        <span class="panel-title">标注管理</span>
        <el-button type="primary" size="small" @click="showCreateForm = true">
          <el-icon><Plus /></el-icon> 新建
        </el-button>
      </div>

      <div class="filter-bar">
        <el-select v-model="priorityFilter" placeholder="优先级" size="small" clearable style="width: 90px" @change="onFilterChange">
          <el-option label="紧急" value="urgent" />
          <el-option label="普通" value="normal" />
          <el-option label="低" value="low" />
        </el-select>
        <el-select v-model="statusFilter" placeholder="状态" size="small" clearable style="width: 90px" @change="onFilterChange">
          <el-option label="打开" value="open" />
          <el-option label="处理中" value="in_progress" />
          <el-option label="已关闭" value="closed" />
        </el-select>
        <el-select v-model="sortBy" placeholder="排序" size="small" style="width: 100px" @change="onSortChange">
          <el-option label="创建时间" value="createdAt" />
          <el-option label="最后回复" value="lastReply" />
        </el-select>
      </div>

      <div class="annotation-items" v-loading="loading">
        <div
          v-for="ann in filteredAnnotations"
          :key="ann.id"
          class="annotation-item"
          :class="{ closed: ann.status === 'closed' }"
          @click="openDetail(ann)"
        >
          <div class="ann-header">
            <span class="priority-dot" :style="{ background: PRIORITY_COLORS[ann.priority] }" />
            <span class="ann-title">{{ ann.title }}</span>
          </div>
          <div class="ann-meta">
            <el-tag size="small" :type="STATUS_TYPES[ann.status]">{{ STATUS_LABELS[ann.status] }}</el-tag>
            <el-tag size="small" :color="PRIORITY_COLORS[ann.priority]" style="color: #fff; border: none">{{ PRIORITY_LABELS[ann.priority] }}</el-tag>
            <span class="ann-type">{{ ann.type === 'element' ? '构件' : '空间' }}</span>
          </div>
          <div class="ann-footer">
            <span class="ann-creator">{{ ann.creator }}</span>
            <span class="ann-comments-count">{{ (ann.comments || []).length }} 条回复</span>
            <span class="ann-time">{{ formatTime(ann.createdAt) }}</span>
          </div>
        </div>
        <div v-if="!loading && filteredAnnotations.length === 0" class="empty-state">
          暂无标注
        </div>
      </div>

      <div class="pagination" v-if="totalPages > 1">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="20"
          :total="total"
          layout="prev, pager, next"
          small
          @current-change="onPageChange"
        />
      </div>
    </div>

    <div v-else class="annotation-detail">
      <div class="detail-header">
        <el-button size="small" @click="closeDetail">
          <el-icon><ArrowLeft /></el-icon> 返回
        </el-button>
        <div class="detail-actions">
          <el-dropdown trigger="click" @command="onStatusCommand">
            <el-button size="small" :type="STATUS_TYPES[currentAnnotation.status]">
              {{ STATUS_LABELS[currentAnnotation.status] }}
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="open">打开</el-dropdown-item>
                <el-dropdown-item command="in_progress">处理中</el-dropdown-item>
                <el-dropdown-item command="closed">已关闭</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-dropdown trigger="click" @command="onPriorityCommand">
            <el-button size="small" :color="PRIORITY_COLORS[currentAnnotation.priority]" style="color: #fff; border: none">
              {{ PRIORITY_LABELS[currentAnnotation.priority] }}
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="urgent">紧急</el-dropdown-item>
                <el-dropdown-item command="normal">普通</el-dropdown-item>
                <el-dropdown-item command="low">低</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button size="small" type="danger" @click="onDelete">删除</el-button>
        </div>
      </div>

      <div class="detail-body">
        <h3 class="detail-title">{{ currentAnnotation.title }}</h3>
        <div class="detail-meta">
          <span class="detail-creator">{{ currentAnnotation.creator }}</span>
          <span class="detail-type">{{ currentAnnotation.type === 'element' ? '构件标注' : '空间标注' }}</span>
          <span class="detail-time">{{ formatTime(currentAnnotation.createdAt) }}</span>
        </div>
        <p class="detail-description">{{ currentAnnotation.description || '无描述' }}</p>

        <div v-if="currentAnnotation.elementId" class="detail-element">
          <el-tag size="small" type="info">构件: {{ currentAnnotation.elementId }}</el-tag>
        </div>

        <div v-if="currentAnnotation.attachments && currentAnnotation.attachments.length > 0" class="detail-attachments">
          <div class="section-title">附件</div>
          <div v-for="att in currentAnnotation.attachments" :key="att.id" class="attachment-item">
            <el-image
              v-if="att.mimeType && att.mimeType.startsWith('image/')"
              :src="`/api/v1/annotations/attachments/${att.filePath}`"
              fit="cover"
              style="width: 120px; height: 90px; border-radius: 4px"
              :preview-src-list="currentAnnotation.attachments.filter(a => a.mimeType && a.mimeType.startsWith('image/')).map(a => `/api/v1/annotations/attachments/${a.filePath}`)"
            />
            <span class="att-name">{{ att.fileName }}</span>
            <span class="att-size">{{ formatSize(att.fileSize) }}</span>
          </div>
        </div>

        <div class="comments-section">
          <div class="section-title">评论 ({{ (currentAnnotation.comments || []).length }})</div>
          <div v-for="comment in currentAnnotation.comments || []" :key="comment.id" class="comment-item">
            <div class="comment-header">
              <span class="comment-author">{{ comment.author }}</span>
              <span class="comment-time">{{ formatTime(comment.createdAt) }}</span>
            </div>
            <p class="comment-content">{{ comment.content }}</p>
            <div v-if="comment.attachment" class="comment-attachment">
              <el-image
                v-if="comment.attachment.mimeType && comment.attachment.mimeType.startsWith('image/')"
                :src="`/api/v1/annotations/attachments/${comment.attachment.filePath}`"
                fit="cover"
                style="width: 100px; height: 75px; border-radius: 4px"
                :preview-src-list="[`/api/v1/annotations/attachments/${comment.attachment.filePath}`]"
              />
              <span class="att-name">{{ comment.attachment.fileName }}</span>
            </div>
          </div>

          <div class="comment-form">
            <el-input
              v-model="newComment"
              type="textarea"
              :rows="2"
              placeholder="添加评论..."
              size="small"
            />
            <div class="comment-form-actions">
              <el-upload
                :auto-upload="false"
                :limit="1"
                accept="image/*"
                :on-change="onCommentFileChange"
                :file-list="commentFileList"
              >
                <el-button size="small">
                  <el-icon><Picture /></el-icon> 图片
                </el-button>
              </el-upload>
              <el-button type="primary" size="small" @click="submitComment" :loading="submitting">
                发送
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <el-dialog v-model="showCreateForm" title="新建标注" width="480px" :close-on-click-modal="false">
      <el-form :model="createForm" label-width="80px" size="small">
        <el-form-item label="类型">
          <el-radio-group v-model="createForm.type">
            <el-radio value="element">构件标注</el-radio>
            <el-radio value="space">空间标注</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="createForm.type === 'element'" label="构件">
          <el-input v-model="createForm.elementId" placeholder="点击构件自动填入" readonly />
        </el-form-item>
        <el-form-item label="标题" required>
          <el-input v-model="createForm.title" placeholder="输入标注标题" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" type="textarea" :rows="3" placeholder="输入描述" />
        </el-form-item>
        <el-form-item label="优先级">
          <el-radio-group v-model="createForm.priority">
            <el-radio value="urgent">紧急</el-radio>
            <el-radio value="normal">普通</el-radio>
            <el-radio value="low">低</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="创建人">
          <el-input v-model="createForm.creator" placeholder="输入姓名" />
        </el-form-item>
        <el-form-item label="附件">
          <el-upload
            :auto-upload="false"
            :limit="3"
            accept="image/*"
            :on-change="onCreateFileChange"
            :file-list="createFileList"
          >
            <el-button size="small">
              <el-icon><Upload /></el-icon> 上传图片 (最多3张)
            </el-button>
          </el-upload>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateForm = false">取消</el-button>
        <el-button type="primary" @click="submitCreate" :loading="creating">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onUnmounted } from 'vue'
import { useAnnotationStore } from '../../stores/annotation'
import { ElMessage, ElMessageBox } from 'element-plus'

const props = defineProps({
  modelId: { type: String, default: null },
  renderer: { type: Object, default: null }
})

const emit = defineEmits(['pin-click', 'create-annotation'])

const store = useAnnotationStore()

const loading = computed(() => store.loading)
const creating = computed(() => store.creating)
const total = computed(() => store.total)
const totalPages = computed(() => store.totalPages)
const filteredAnnotations = computed(() => store.filteredAnnotations)
const currentAnnotation = computed(() => store.currentAnnotation)

const PRIORITY_COLORS = store.PRIORITY_COLORS
const PRIORITY_LABELS = store.PRIORITY_LABELS
const STATUS_LABELS = store.STATUS_LABELS
const STATUS_TYPES = store.STATUS_TYPES

const priorityFilter = ref('')
const statusFilter = ref('')
const sortBy = ref('createdAt')
const currentPage = ref(1)

const showCreateForm = ref(false)
const createForm = ref({
  type: 'element',
  elementId: '',
  position: [0, 0, 0],
  title: '',
  description: '',
  priority: 'normal',
  creator: ''
})
const createFileList = ref([])

const newComment = ref('')
const commentFileList = ref([])
const submitting = ref(false)

watch(() => props.modelId, (id) => {
  if (id) {
    store.fetchAnnotations(id, true)
    store.connectWebSocket(id)
  }
}, { immediate: true })

watch(() => store.annotations, (anns) => {
  if (!props.renderer) return
  props.renderer.clearAnnotationPins()
  for (const ann of anns) {
    props.renderer.addAnnotationPin(ann)
  }
}, { deep: true })

watch(() => props.renderer, (r) => {
  if (!r) return
  r.onAnnotationClick = (pinId) => {
    const ann = store.annotations.find(a => a.id === pinId)
    if (ann) {
      store.setCurrentAnnotation(ann)
      emit('pin-click', ann)
    }
  }
  r.onAnnotationDblClick = (pinId, elementId, position) => {
    if (pinId) {
      const ann = store.annotations.find(a => a.id === pinId)
      if (ann) {
        store.setCurrentAnnotation(ann)
        emit('pin-click', ann)
      }
    } else {
      showCreateForm.value = true
      if (elementId) {
        createForm.value.type = 'element'
        createForm.value.elementId = elementId
        createForm.value.position = [position.x, position.y, position.z]
      } else {
        createForm.value.type = 'space'
        createForm.value.elementId = ''
        createForm.value.position = [position.x, position.y, position.z]
      }
    }
  }
})

onUnmounted(() => {
  store.disconnectWebSocket()
  if (props.renderer) {
    props.renderer.clearAnnotationPins()
    props.renderer.onAnnotationClick = null
    props.renderer.onAnnotationDblClick = null
  }
})

function onFilterChange() {
  store.setPriorityFilter(priorityFilter.value)
  store.setStatusFilter(statusFilter.value)
  if (props.modelId) store.fetchAnnotations(props.modelId, true)
}

function onSortChange() {
  store.setSortBy(sortBy.value)
  if (props.modelId) store.fetchAnnotations(props.modelId, true)
}

function onPageChange(p) {
  currentPage.value = p
  store.setPage(p)
  if (props.modelId) store.fetchAnnotations(props.modelId)
}

async function openDetail(ann) {
  await store.fetchAnnotation(ann.id)
  emit('pin-click', ann)
}

function closeDetail() {
  store.setCurrentAnnotation(null)
}

async function onStatusCommand(status) {
  if (!currentAnnotation.value) return
  try {
    await store.updateAnnotation(currentAnnotation.value.id, { status })
    if (props.renderer) {
      props.renderer.updateAnnotationPin(currentAnnotation.value)
    }
    ElMessage.success('状态已更新')
  } catch (e) {
    ElMessage.error('更新失败')
  }
}

async function onPriorityCommand(priority) {
  if (!currentAnnotation.value) return
  try {
    await store.updateAnnotation(currentAnnotation.value.id, { priority })
    if (props.renderer) {
      props.renderer.updateAnnotationPin(currentAnnotation.value)
    }
    ElMessage.success('优先级已更新')
  } catch (e) {
    ElMessage.error('更新失败')
  }
}

async function onDelete() {
  if (!currentAnnotation.value) return
  try {
    await ElMessageBox.confirm('确定要删除此标注吗？', '确认删除', { type: 'warning' })
    const id = currentAnnotation.value.id
    await store.deleteAnnotation(id)
    if (props.renderer) {
      props.renderer.removeAnnotationPin(id)
    }
    closeDetail()
    ElMessage.success('已删除')
  } catch (e) {
    // cancelled
  }
}

function onCreateFileChange(file, fileList) {
  createFileList.value = fileList.slice(-3)
}

async function submitCreate() {
  if (!createForm.value.title) {
    ElMessage.warning('请输入标题')
    return
  }
  if (!props.modelId) {
    ElMessage.warning('未选择模型')
    return
  }

  const formData = new FormData()
  formData.append('modelId', props.modelId)
  formData.append('type', createForm.value.type)
  formData.append('title', createForm.value.title)
  formData.append('description', createForm.value.description || '')
  formData.append('priority', createForm.value.priority)
  formData.append('creator', createForm.value.creator || '匿名')
  formData.append('position', JSON.stringify(createForm.value.position))

  if (createForm.value.type === 'element' && createForm.value.elementId) {
    formData.append('elementId', createForm.value.elementId)
  }

  for (const f of createFileList.value) {
    formData.append('attachments', f.raw)
  }

  try {
    const ann = await store.createAnnotation(formData)
    if (props.renderer && ann) {
      props.renderer.addAnnotationPin(ann)
    }
    showCreateForm.value = false
    createForm.value = {
      type: 'element',
      elementId: '',
      position: [0, 0, 0],
      title: '',
      description: '',
      priority: 'normal',
      creator: ''
    }
    createFileList.value = []
    ElMessage.success('标注已创建')
  } catch (e) {
    ElMessage.error('创建失败')
  }
}

function onCommentFileChange(file, fileList) {
  commentFileList.value = fileList.slice(-1)
}

async function submitComment() {
  if (!newComment.value.trim()) {
    ElMessage.warning('请输入评论内容')
    return
  }
  if (!currentAnnotation.value) return

  submitting.value = true
  const formData = new FormData()
  formData.append('content', newComment.value)
  formData.append('author', '匿名')

  if (commentFileList.value.length > 0) {
    formData.append('attachment', commentFileList.value[0].raw)
  }

  try {
    await store.addComment(currentAnnotation.value.id, formData)
    newComment.value = ''
    commentFileList.value = []
    ElMessage.success('评论已发送')
  } catch (e) {
    ElMessage.error('评论失败')
  } finally {
    submitting.value = false
  }
}

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  const now = new Date()
  const diff = now - d
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  const pad = n => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function formatSize(bytes) {
  if (!bytes) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1024 / 1024).toFixed(1) + ' MB'
}
</script>

<style scoped>
.annotation-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  color: #c0c4cc;
  font-size: 13px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-bottom: 1px solid #2a2a4a;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
  color: #e0e0e0;
}

.filter-bar {
  display: flex;
  gap: 6px;
  padding: 8px 12px;
  border-bottom: 1px solid #2a2a4a;
  flex-wrap: wrap;
}

.annotation-items {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.annotation-item {
  background: #1e2a45;
  border-radius: 6px;
  padding: 10px 12px;
  margin-bottom: 6px;
  cursor: pointer;
  transition: background 0.2s;
  border: 1px solid transparent;
}

.annotation-item:hover {
  background: #243050;
  border-color: #409eff33;
}

.annotation-item.closed {
  opacity: 0.6;
}

.ann-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.priority-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.ann-title {
  font-weight: 500;
  color: #e0e0e0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.ann-meta {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-bottom: 6px;
}

.ann-type {
  font-size: 11px;
  color: #8899aa;
}

.ann-footer {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: #667788;
}

.ann-creator {
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-state {
  text-align: center;
  padding: 40px 0;
  color: #667788;
}

.pagination {
  display: flex;
  justify-content: center;
  padding: 8px;
  border-top: 1px solid #2a2a4a;
}

.annotation-detail {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-bottom: 1px solid #2a2a4a;
}

.detail-actions {
  display: flex;
  gap: 6px;
}

.detail-body {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.detail-title {
  font-size: 16px;
  font-weight: 600;
  color: #e0e0e0;
  margin: 0 0 8px 0;
}

.detail-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: #8899aa;
  margin-bottom: 12px;
}

.detail-description {
  color: #aabbcc;
  line-height: 1.6;
  margin: 0 0 12px 0;
}

.detail-element {
  margin-bottom: 12px;
}

.detail-attachments {
  margin-bottom: 16px;
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: #c0c4cc;
  margin-bottom: 8px;
}

.attachment-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.att-name {
  color: #8899aa;
  font-size: 12px;
}

.att-size {
  color: #667788;
  font-size: 11px;
}

.comments-section {
  border-top: 1px solid #2a2a4a;
  padding-top: 12px;
}

.comment-item {
  background: #1e2a45;
  border-radius: 6px;
  padding: 8px 10px;
  margin-bottom: 6px;
}

.comment-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 4px;
}

.comment-author {
  color: #409eff;
  font-size: 12px;
  font-weight: 500;
}

.comment-time {
  color: #667788;
  font-size: 11px;
}

.comment-content {
  color: #c0c4cc;
  font-size: 13px;
  margin: 4px 0 0 0;
  line-height: 1.5;
}

.comment-attachment {
  margin-top: 6px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.comment-form {
  margin-top: 12px;
}

.comment-form-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 6px;
}
</style>
