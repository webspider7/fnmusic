<template>
  <div class="relative w-full h-full flex flex-col justify-end overflow-hidden select-none">
    <!-- Visualizer Mode Pills -->
    <div class="absolute top-3 right-3 z-10 flex items-center gap-1.5 p-1 rounded-xl bg-dark-900/70 backdrop-blur-md border border-white/10 text-xs">
      <button 
        v-for="m in modes" 
        :key="m.id"
        @click="switchMode(m.id)"
        :class="[
          'px-2.5 py-1 rounded-lg transition-all flex items-center gap-1 font-medium',
          currentMode === m.id 
            ? 'bg-gradient-to-r from-aurora-cyan to-aurora-violet text-dark-950 font-bold shadow-md' 
            : 'text-slate-400 hover:text-white hover:bg-white/5'
        ]"
      >
        <component :is="m.icon" class="w-3.5 h-3.5" />
        <span>{{ m.name }}</span>
      </button>
    </div>

    <!-- Canvas -->
    <canvas ref="canvasRef" class="w-full h-full block"></canvas>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { AudioVisualizerEngine } from '../engine/audio-visualizer'
import { BarChart3, Activity, CircleDot } from 'lucide-vue-next'

const props = defineProps({
  audioElement: Object,
  isPlaying: Boolean,
})

const canvasRef = ref(null)
let engine = null
const currentMode = ref('bars')

const modes = [
  { id: 'bars', name: '能量柱', icon: BarChart3 },
  { id: 'wave', name: '霓虹波', icon: Activity },
  { id: 'reactor', name: '反应堆', icon: CircleDot },
]

function switchMode(mode) {
  currentMode.value = mode
  if (engine) engine.setMode(mode)
}

onMounted(() => {
  if (canvasRef.value && props.audioElement) {
    engine = new AudioVisualizerEngine(canvasRef.value, props.audioElement)
    engine.setMode(currentMode.value)
    engine.start()
  }
})

watch(() => props.isPlaying, (playing) => {
  if (engine) {
    if (playing) {
      engine.resume()
      engine.start()
    }
  }
})

onUnmounted(() => {
  if (engine) engine.stop()
})
</script>
