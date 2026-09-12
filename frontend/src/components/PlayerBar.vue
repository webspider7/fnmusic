<template>
  <div class="h-[68px] bg-white border-t border-gray-100 shadow-sm px-3 sm:px-5 flex items-center justify-between select-none z-40 shrink-0">
    <!-- Left: Cover + Info -->
    <div class="flex items-center gap-2.5 sm:gap-3 max-w-[150px] sm:w-1/4 sm:min-w-[180px] shrink-0 cursor-pointer" @click="$emit('toggle-fullscreen')">
      <div class="relative w-10 h-10 rounded-xl overflow-hidden bg-gray-100 shrink-0">
        <img v-if="currentSong && currentSong.cover" :src="currentSong.cover" alt="cover" referrerpolicy="no-referrer" class="w-full h-full object-cover" @error="currentSong.cover = ''" />
        <div v-else class="w-full h-full flex items-center justify-center">
          <Music class="w-4 h-4 text-emerald-400" />
        </div>
      </div>
      <div class="overflow-hidden min-w-0">
        <p class="text-xs sm:text-sm font-semibold text-gray-800 truncate max-w-[90px] sm:max-w-[140px]">{{ currentSong ? (currentSong.name || currentSong.title || '极光音乐') : '极光音乐' }}</p>
        <p class="text-[10px] sm:text-xs text-gray-400 truncate max-w-[90px] sm:max-w-[140px]">{{ currentSong ? (currentSong.singer || currentSong.artist || '探索无限声场') : '探索无限声场' }}</p>
      </div>
    </div>

    <!-- Center: Controls + Progress -->
    <div class="flex-1 max-w-xl flex flex-col items-center gap-1 px-2 sm:px-4 min-w-0">
      <div class="flex items-center gap-2 sm:gap-4">
        <button @click="cyclePlayMode" :title="playModeTitle" class="text-gray-400 hover:text-gray-700 transition-colors p-1">
          <Shuffle v-if="playMode === 'random'" class="w-3.5 h-3.5 text-emerald-500" />
          <Repeat1 v-else-if="playMode === 'loop'" class="w-3.5 h-3.5 text-emerald-500" />
          <Repeat v-else class="w-3.5 h-3.5" />
        </button>
        <button @click="$emit('prev')" class="text-gray-400 hover:text-gray-700 transition-colors p-1" title="上一首">
          <SkipBack class="w-4 h-4 fill-current" />
        </button>
        <button @click="$emit('toggle-play')" :disabled="isResolving"
          class="w-8 h-8 sm:w-9 sm:h-9 rounded-full bg-emerald-500 hover:bg-emerald-600 text-white flex items-center justify-center shadow-sm transition-all active:scale-95 shrink-0"
        >
          <Loader2 v-if="isResolving" class="w-4 h-4 animate-spin" />
          <Pause v-else-if="isPlaying" class="w-4 h-4 fill-current" />
          <Play v-else class="w-4 h-4 fill-current translate-x-0.5" />
        </button>
        <button @click="$emit('next')" class="text-gray-400 hover:text-gray-700 transition-colors p-1" title="下一首">
          <SkipForward class="w-4 h-4 fill-current" />
        </button>
        <button @click="$emit('toggle-lyrics')"
          :class="['text-xs font-semibold px-1.5 py-0.5 rounded transition-colors', showLyrics ? 'text-emerald-600 bg-emerald-50' : 'text-gray-400 hover:text-gray-600']"
        >词</button>
      </div>
      <div class="w-full flex items-center gap-1.5 sm:gap-2 text-[10px] text-gray-400 font-mono">
        <span class="w-8 sm:w-9 text-right shrink-0">{{ formatTime(currentTime) }}</span>
        <div ref="progressBarRef" @click="handleSeek" class="flex-1 h-1.5 bg-gray-200 rounded-full relative cursor-pointer group">
          <div class="h-1.5 bg-emerald-500 rounded-full transition-all relative" :style="{ width: progressPercent + '%' }">
            <div class="absolute right-0 top-1/2 -translate-y-1/2 w-2.5 h-2.5 rounded-full bg-emerald-500 shadow-sm opacity-0 group-hover:opacity-100 transition-opacity"></div>
          </div>
        </div>
        <span class="w-8 sm:w-9 shrink-0">{{ formatTime(duration) }}</span>
      </div>
    </div>

    <!-- Right: Volume (hidden on mobile) -->
    <div class="hidden md:flex items-center justify-end gap-3 w-1/4 min-w-[160px]">
      <div v-if="isPlaying" class="flex items-end gap-0.5 h-4">
        <span class="w-0.5 bg-emerald-400 rounded-t animate-bounce" style="height:14px;animation-duration:.6s"></span>
        <span class="w-0.5 bg-teal-400 rounded-t animate-bounce" style="height:8px;animation-duration:.4s"></span>
        <span class="w-0.5 bg-sky-400 rounded-t animate-bounce" style="height:12px;animation-duration:.5s"></span>
      </div>
      <button @click="toggleMute" class="text-gray-400 hover:text-gray-700 transition-colors">
        <VolumeX v-if="isMuted" class="w-4 h-4 text-red-400" />
        <Volume1 v-else-if="volume < 0.5" class="w-4 h-4" />
        <Volume2 v-else class="w-4 h-4" />
      </button>
      <input type="range" min="0" max="1" step="0.01" v-model="volume"
        class="w-20 h-1 rounded-full appearance-none cursor-pointer accent-emerald-500 bg-gray-200" />
      <button @click="$emit('toggle-fullscreen')" class="text-gray-400 hover:text-gray-700 transition-colors" title="全屏">
        <Maximize class="w-4 h-4" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Play, Pause, SkipBack, SkipForward, Shuffle, Repeat, Repeat1, Volume2, Volume1, VolumeX, Maximize, Music, Loader2 } from 'lucide-vue-next'

const props = defineProps({
  currentSong: Object,
  isPlaying: Boolean,
  currentTime: Number,
  duration: Number,
  showLyrics: Boolean,
  isResolving: Boolean,
  playMode: { type: String, default: 'sequence' },
})
const emit = defineEmits(['toggle-play', 'prev', 'next', 'seek', 'toggle-lyrics', 'toggle-fullscreen', 'toggle-view', 'volume-change', 'update:playMode'])

const volume = ref(0.8)
const isMuted = ref(false)
const prevVolume = ref(0.8)
const progressBarRef = ref(null)

const playModes = ['sequence', 'random', 'loop']
const playModeTitle = computed(() => ({ sequence: '列表循环', random: '随机播放', loop: '单曲循环' }[props.playMode]))
function cyclePlayMode() {
  const nextIdx = (playModes.indexOf(props.playMode) + 1) % playModes.length
  emit('update:playMode', playModes[nextIdx])
}

const progressPercent = computed(() => !props.duration ? 0 : Math.min(100, (props.currentTime / props.duration) * 100))

function formatTime(sec) {
  if (!sec || isNaN(sec)) return '00:00'
  return Math.floor(sec / 60).toString().padStart(2, '0') + ':' + Math.floor(sec % 60).toString().padStart(2, '0')
}

function handleSeek(e) {
  if (!progressBarRef.value || !props.duration) return
  const rect = progressBarRef.value.getBoundingClientRect()
  emit('seek', Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width)) * props.duration)
}

function toggleMute() {
  if (isMuted.value) { isMuted.value = false; volume.value = prevVolume.value || 0.8 }
  else { prevVolume.value = volume.value; volume.value = 0; isMuted.value = true }
}

watch(volume, (val) => { emit('volume-change', parseFloat(val)); if (val > 0) isMuted.value = false })
</script>
