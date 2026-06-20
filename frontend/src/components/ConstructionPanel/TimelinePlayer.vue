<template>
  <div class="timeline-player" v-if="constructionStore.playbackActive && constructionStore.currentPlan">
    <div class="player-controls">
      <el-button
        size="small"
        circle
        :type="constructionStore.playing ? 'warning' : 'success'"
        @click="togglePlay"
      >
        <el-icon>
          <VideoPlay v-if="!constructionStore.playing" />
          <VideoPause v-else />
        </el-icon>
      </el-button>

      <div class="slider-area" ref="sliderRef" @mousedown="onSliderMouseDown">
        <div class="slider-track">
          <div class="slider-fill" :style="{ width: (constructionStore.playProgress * 100) + '%' }" />
          <div class="slider-thumb" :style="{ left: (constructionStore.playProgress * 100) + '%' }" />
        </div>
        <div class="slider-ticks">
          <div
            v-for="(phase, idx) in constructionStore.allPhases"
            :key="phase.id"
            class="phase-tick"
            :style="{
              left: getPhaseTickPos(phase.startDate) + '%',
              width: getPhaseTickWidth(phase) + '%',
              background: phase.color || constructionStore.PHASE_COLORS[idx % constructionStore.PHASE_COLORS.length]
            }"
          />
        </div>
      </div>

      <div class="speed-select">
        <el-select v-model="speedValue" size="small" style="width: 70px" @change="onSpeedChange">
          <el-option v-for="s in constructionStore.SPEED_OPTIONS" :key="s" :label="s + 'x'" :value="s" />
        </el-select>
      </div>

      <div class="current-date">
        {{ constructionStore.currentDate || constructionStore.currentPlan.startDate }}
      </div>

      <el-button size="small" type="danger" @click="stopPlayback">
        退出
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onUnmounted } from 'vue'
import { useConstructionStore } from '../../stores/construction'
import { VideoPlay, VideoPause } from '@element-plus/icons-vue'

const constructionStore = useConstructionStore()
const sliderRef = ref(null)
const speedValue = ref(1)
const isDragging = ref(false)
let animationFrameId = null
let lastTime = null

const BASE_DAYS_PER_SECOND = 5

function togglePlay() {
  if (constructionStore.playing) {
    constructionStore.pausePlayback()
  } else {
    constructionStore.startPlayback()
  }
}

function stopPlayback() {
  constructionStore.stopPlayback()
  if (animationFrameId) {
    cancelAnimationFrame(animationFrameId)
    animationFrameId = null
  }
  lastTime = null
}

function onSpeedChange(speed) {
  constructionStore.setPlaySpeed(speed)
}

function getPhaseTickPos(dateStr) {
  const plan = constructionStore.currentPlan
  if (!plan) return 0
  const start = new Date(plan.startDate).getTime()
  const end = new Date(plan.endDate).getTime()
  const d = new Date(dateStr).getTime()
  return ((d - start) / (end - start)) * 100
}

function getPhaseTickWidth(phase) {
  const plan = constructionStore.currentPlan
  if (!plan) return 0
  const start = new Date(plan.startDate).getTime()
  const end = new Date(plan.endDate).getTime()
  const phaseStart = new Date(phase.startDate).getTime()
  const phaseEnd = new Date(phase.endDate).getTime()
  return ((phaseEnd - phaseStart) / (end - start)) * 100
}

function onSliderMouseDown(e) {
  isDragging.value = true
  updateProgressFromEvent(e)

  const onMouseMove = (e) => {
    if (isDragging.value) {
      updateProgressFromEvent(e)
    }
  }

  const onMouseUp = () => {
    isDragging.value = false
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }

  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

function updateProgressFromEvent(e) {
  if (!sliderRef.value) return
  const rect = sliderRef.value.getBoundingClientRect()
  const x = e.clientX - rect.left
  const progress = Math.max(0, Math.min(1, x / rect.width))
  constructionStore.seekToProgress(progress)
}

function playbackLoop(timestamp) {
  if (!constructionStore.playing) {
    lastTime = null
    animationFrameId = requestAnimationFrame(playbackLoop)
    return
  }

  if (lastTime === null) {
    lastTime = timestamp
    animationFrameId = requestAnimationFrame(playbackLoop)
    return
  }

  const deltaMs = timestamp - lastTime
  lastTime = timestamp

  const deltaDays = (deltaMs / 1000) * BASE_DAYS_PER_SECOND * constructionStore.playSpeed
  constructionStore.advancePlayback(deltaDays)

  if (!constructionStore.playing) {
    lastTime = null
  }

  animationFrameId = requestAnimationFrame(playbackLoop)
}

watch(() => constructionStore.playbackActive, (active) => {
  if (active) {
    lastTime = null
    animationFrameId = requestAnimationFrame(playbackLoop)
  } else {
    if (animationFrameId) {
      cancelAnimationFrame(animationFrameId)
      animationFrameId = null
    }
    lastTime = null
  }
}, { immediate: true })

onUnmounted(() => {
  if (animationFrameId) {
    cancelAnimationFrame(animationFrameId)
    animationFrameId = null
  }
})
</script>

<style scoped>
.timeline-player {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: rgba(22, 33, 62, 0.97);
  border-top: 1px solid #2a2a4a;
  z-index: 20;
  padding: 10px 16px;
}

.player-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.slider-area {
  flex: 1;
  position: relative;
  height: 32px;
  cursor: pointer;
  padding: 14px 0;
}

.slider-track {
  position: absolute;
  top: 14px;
  left: 0;
  right: 0;
  height: 4px;
  background: #2a2a4a;
  border-radius: 2px;
}

.slider-fill {
  position: absolute;
  top: 0;
  left: 0;
  height: 100%;
  background: #409EFF;
  border-radius: 2px;
}

.slider-thumb {
  position: absolute;
  top: 50%;
  width: 12px;
  height: 12px;
  background: #ffffff;
  border: 2px solid #409EFF;
  border-radius: 50%;
  transform: translate(-50%, -50%);
  z-index: 2;
  transition: transform 0.1s;
}

.slider-thumb:hover {
  transform: translate(-50%, -50%) scale(1.3);
}

.slider-ticks {
  position: absolute;
  top: 20px;
  left: 0;
  right: 0;
  height: 3px;
}

.phase-tick {
  position: absolute;
  top: 0;
  height: 100%;
  border-radius: 1px;
  opacity: 0.6;
  min-width: 1px;
}

.speed-select {
  flex-shrink: 0;
}

.current-date {
  font-size: 13px;
  color: #ffffff;
  font-family: monospace;
  min-width: 100px;
  text-align: center;
  flex-shrink: 0;
  background: rgba(0, 0, 0, 0.3);
  padding: 4px 8px;
  border-radius: 4px;
}
</style>
