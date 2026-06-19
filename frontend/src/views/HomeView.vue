<template>
  <div class="home-view">
    <div class="hero">
      <h1>BIM Viewer</h1>
      <p>轻量化BIM模型浏览与碰撞检测工具</p>
    </div>

    <div class="upload-section">
      <el-upload
        drag
        accept=".ifc"
        :auto-upload="false"
        :show-file-list="false"
        :on-change="onFileChange"
      >
        <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
        <div class="el-upload__text">
          拖拽IFC文件到此处，或 <em>点击上传</em>
        </div>
        <template #tip>
          <div class="el-upload__tip">支持 IFC 2x3 和 IFC4 格式，最大 500MB</div>
        </template>
      </el-upload>

      <el-progress
        v-if="uploadProgress > 0 && uploadProgress < 100"
        :percentage="uploadProgress"
        :stroke-width="4"
        style="margin-top: 16px"
      />

      <el-alert
        v-if="uploadError"
        :title="uploadError"
        type="error"
        show-icon
        closable
        style="margin-top: 12px"
      />
    </div>

    <div class="models-section">
      <h2>模型列表</h2>
      <el-table
        :data="modelStore.models"
        v-loading="modelStore.loading"
        empty-text="暂无模型，请上传IFC文件"
        @row-click="openModel"
        style="cursor: pointer"
      >
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="ifcVersion" label="IFC版本" width="100" />
        <el-table-column prop="fileSize" label="文件大小" width="120">
          <template #default="{ row }">
            {{ formatFileSize(row.fileSize) }}
          </template>
        </el-table-column>
        <el-table-column prop="triangleCount" label="三角面数" width="120">
          <template #default="{ row }">
            {{ row.triangleCount?.toLocaleString() || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="elementCount" label="构件数" width="100">
          <template #default="{ row }">
            {{ row.elementCount?.toLocaleString() || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.status === 'ready' ? 'success' : row.status === 'parsing' ? 'warning' : 'danger'"
              size="small"
            >
              {{ row.status === 'ready' ? '就绪' : row.status === 'parsing' ? '解析中' : row.status === 'error' ? '错误' : '上传中' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click.stop="openModel(row)">打开</el-button>
            <el-button size="small" type="danger" @click.stop="deleteModel(row.id)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useModelStore } from '../stores/model'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const modelStore = useModelStore()
const uploadProgress = ref(0)
const uploadError = ref('')

onMounted(() => {
  modelStore.fetchModels()
})

async function onFileChange(uploadFile) {
  if (!uploadFile.raw) return
  uploadError.value = ''
  uploadProgress.value = 0

  try {
    const model = await modelStore.uploadModel(uploadFile.raw, (progress) => {
      uploadProgress.value = progress
    })
    ElMessage.success('模型上传并解析成功')
    uploadProgress.value = 100
    router.push(`/viewer/${model.id}`)
  } catch (err) {
    uploadError.value = err.response?.data?.error || '上传失败'
    uploadProgress.value = 0
  }
}

function openModel(model) {
  if (model.status === 'ready') {
    router.push(`/viewer/${model.id}`)
  } else {
    ElMessage.warning('模型尚未就绪')
  }
}

async function deleteModel(modelId) {
  try {
    await ElMessageBox.confirm('确定要删除此模型吗？', '确认删除', {
      type: 'warning'
    })
    await modelStore.deleteModel(modelId)
    ElMessage.success('模型已删除')
  } catch {
    // cancelled
  }
}

function formatFileSize(bytes) {
  if (!bytes) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}
</script>

<style scoped>
.home-view {
  min-height: 100vh;
  background: #0f0f23;
  color: #ccddee;
  padding: 40px;
}

.hero {
  text-align: center;
  margin-bottom: 40px;
}

.hero h1 {
  font-size: 36px;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 8px;
}

.hero p {
  font-size: 16px;
  color: #8899aa;
}

.upload-section {
  max-width: 600px;
  margin: 0 auto 40px;
}

:deep(.el-upload-dragger) {
  background: #1a1a3e;
  border-color: #3a3a6a;
}

:deep(.el-upload-dragger:hover) {
  border-color: #4080ff;
}

.models-section {
  max-width: 900px;
  margin: 0 auto;
}

.models-section h2 {
  font-size: 18px;
  margin-bottom: 16px;
  color: #ccddee;
}

:deep(.el-table) {
  background: transparent;
  --el-table-bg-color: transparent;
  --el-table-tr-bg-color: transparent;
  --el-table-header-bg-color: rgba(42, 42, 74, 0.5);
  --el-table-row-hover-bg-color: rgba(64, 128, 255, 0.1);
  --el-table-border-color: #2a2a4a;
  --el-table-text-color: #aabbcc;
  --el-table-header-text-color: #8899aa;
}
</style>
