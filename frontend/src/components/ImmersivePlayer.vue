<template>
  <div class="fixed inset-0 z-50 flex flex-col bg-[#1a1c26] text-white select-none overflow-hidden">

    <!-- TOP BAR -->
    <div class="h-14 flex items-center justify-between px-3 md:px-5 shrink-0 border-b border-white/5 relative">
      <button @click="$emit('collapse')"
        class="flex items-center gap-1 px-2.5 py-1.5 rounded-xl hover:bg-white/8 text-gray-300 hover:text-white transition-all text-xs shrink-0">
        <ChevronDown class="w-4 h-4" />
        <span class="hidden sm:inline">收起</span>
      </button>

      <!-- 移动端视图切换 Tabs (只在 < md 显示) -->
      <div class="flex md:hidden items-center bg-white/10 rounded-xl p-0.5">
        <button @click="mobileTab = 'cover'"
          :class="['px-3 py-1 rounded-lg text-xs font-medium transition-all', mobileTab === 'cover' ? 'bg-emerald-500 text-white' : 'text-gray-300']">
          唱盘
        </button>
        <button @click="mobileTab = 'lyrics'"
          :class="['px-3 py-1 rounded-lg text-xs font-medium transition-all', mobileTab === 'lyrics' ? 'bg-emerald-500 text-white' : 'text-gray-300']">
          歌词
        </button>
      </div>

      <!-- 桌面端居中标题 -->
      <div class="hidden md:block text-center absolute left-1/2 -translate-x-1/2">
        <p class="text-[11px] text-emerald-400 font-medium mb-0.5">● 正在播放</p>
        <p class="text-sm font-bold text-white truncate max-w-xs">{{ currentSong?.name || '—' }}</p>
      </div>

      <div class="flex items-center gap-1.5 md:gap-2 shrink-0">
        <button @click="showTranslation = !showTranslation"
          :class="['px-2 md:px-2.5 py-1.5 rounded-xl text-xs font-medium transition-all flex items-center gap-1',
            showTranslation ? 'bg-emerald-500/20 text-emerald-400' : 'hover:bg-white/8 text-gray-400']">
          <AlignCenter class="w-3.5 h-3.5" />
          <span class="hidden sm:inline">翻译</span>
        </button>
        <div class="hidden sm:flex items-center gap-1 bg-white/5 rounded-xl px-1.5 py-1">
          <button @click="lyricFontSize = Math.max(13, lyricFontSize - 2)" class="w-6 h-6 flex items-center justify-center hover:bg-white/10 rounded-lg text-gray-400 text-xs font-bold">A</button>
          <div class="w-px h-4 bg-white/15"></div>
          <button @click="lyricFontSize = Math.min(28, lyricFontSize + 2)" class="w-6 h-6 flex items-center justify-center hover:bg-white/10 rounded-lg text-gray-300 text-sm font-bold">A</button>
        </div>
      </div>
    </div>

    <!-- MAIN CONTENT -->
    <div class="flex-1 flex flex-col md:flex-row overflow-hidden min-h-0">

      <!-- LEFT: Vinyl + Info + Actions (md:w-[42%], hidden on mobile if mobileTab === 'lyrics') -->
      <div :class="['w-full md:w-[42%] flex flex-col items-center justify-center p-4 md:px-12 md:py-6 gap-4 md:gap-6 shrink-0', mobileTab === 'lyrics' ? 'hidden md:flex' : 'flex flex-1']">

        <!-- Vinyl Disc -->
        <div class="relative">
          <!-- Tonearm -->
          <div :class="['absolute z-10 transition-transform duration-700', isPlaying ? 'rotate-[28deg]' : 'rotate-0']"
            style="top:-16px; right:-8px; transform-origin:16px 16px">
            <svg width="80" height="80" viewBox="0 0 80 80" fill="none">
              <line x1="16" y1="16" x2="56" y2="60" stroke="#999" stroke-width="2.5" stroke-linecap="round"/>
              <circle cx="56" cy="60" r="5" fill="#777" stroke="#bbb" stroke-width="1.5"/>
              <circle cx="16" cy="16" r="7" fill="#555" stroke="#888" stroke-width="1.5"/>
            </svg>
          </div>
          <!-- Disc -->
          <div :class="['w-48 h-48 md:w-56 md:h-56 rounded-full shadow-2xl relative', isPlaying ? 'animate-spin' : '']"
            style="animation-duration:18s; background:#111; border:4px solid #2a2d3e">
            <!-- Groove rings -->
            <div class="absolute inset-0 rounded-full" style="background:repeating-radial-gradient(circle,transparent 0,transparent 6px,rgba(255,255,255,.03) 6px,rgba(255,255,255,.03) 7px)"></div>
            <!-- Album cover -->
            <div class="absolute inset-5 md:inset-6 rounded-full overflow-hidden border border-white/10">
              <img v-if="currentSong?.cover" :src="currentSong.cover" alt="cover" referrerpolicy="no-referrer" class="w-full h-full object-cover" @error="$event.target.style.display='none'" />
              <div v-else class="w-full h-full bg-gradient-to-br from-emerald-800/50 to-teal-900/50 flex items-center justify-center">
                <Music class="w-10 h-10 text-emerald-400/60" />
              </div>
            </div>
            <!-- Center hole -->
            <div class="absolute inset-0 flex items-center justify-center pointer-events-none">
              <div class="w-3 h-3 rounded-full bg-[#1a1c26] border border-white/20 z-10"></div>
            </div>
          </div>
        </div>

        <!-- Song Info -->
        <div class="text-center px-4">
          <div class="flex items-center justify-center gap-2 mb-1">
            <h2 class="text-lg md:text-xl font-bold text-white truncate max-w-[240px] md:max-w-[260px]">{{ currentSong?.name || '—' }}</h2>
            <span v-if="currentSong?.quality" class="text-[10px] px-1.5 py-0.5 rounded border border-emerald-500/50 text-emerald-400 font-mono shrink-0">{{ currentSong.quality }}</span>
          </div>
          <p class="text-xs md:text-sm text-gray-400">{{ currentSong?.singer || '—' }}</p>
        </div>

        <!-- Actions -->
        <div class="flex items-center gap-2 flex-wrap justify-center">
          <button @click="copyLyrics" class="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white/5 hover:bg-white/10 text-gray-300 text-xs font-semibold border border-white/8 transition-all">
            <Copy class="w-3.5 h-3.5" />复制歌词
          </button>
        </div>
      </div>

      <!-- RIGHT: Lyrics (hidden on mobile if mobileTab === 'cover') -->
      <div :class="['flex-1 flex flex-col overflow-hidden border-t md:border-t-0 md:border-l border-white/5', mobileTab === 'cover' ? 'hidden md:flex' : 'flex']">
        <div v-if="!parsedLyrics.length" class="flex-1 flex flex-col items-center justify-center text-gray-600">
          <Music class="w-12 h-12 mb-3 opacity-30" />
          <p class="text-sm">享受动人旋律...</p>
        </div>
        <div v-else ref="lyricContainer"
          class="flex-1 overflow-y-auto overflow-x-hidden py-16 md:py-32 px-4 md:px-12 text-center space-y-4 md:space-y-5 scroll-smooth"
          style="scrollbar-width:none">
          <div
            v-for="(line, idx) in parsedLyrics"
            :key="idx"
            :ref="el => setLineRef(el, idx)"
            @click="$emit('seek', line.time)"
            :class="[
              'transition-all duration-300 cursor-pointer leading-relaxed',
              currentLineIndex === idx
                ? 'text-emerald-400 font-bold'
                : 'text-gray-500 hover:text-gray-300'
            ]"
            :style="{ fontSize: (currentLineIndex === idx ? lyricFontSize + 2 : lyricFontSize) + 'px' }"
          >
            <p>{{ line.text }}</p>
            <p v-if="showTranslation && line.trans" class="text-xs mt-0.5 opacity-70">{{ line.trans }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- BOTTOM CONTROLS -->
    <div class="h-20 md:h-24 border-t border-white/5 px-4 md:px-8 flex flex-col justify-center gap-2 md:gap-3 shrink-0">
      <!-- Progress -->
      <div class="flex items-center gap-2 md:gap-3 text-[11px] text-gray-400 font-mono">
        <span class="w-9 md:w-10 text-right shrink-0">{{ formatTime(currentTime) }}</span>
        <div ref="progressBarRef" @click="handleSeek"
          class="flex-1 h-1.5 bg-white/10 rounded-full cursor-pointer group relative">
          <div class="h-1.5 bg-emerald-500 rounded-full relative" :style="{ width: progressPercent + '%' }">
            <div class="absolute right-0 top-1/2 -translate-y-1/2 w-3.5 h-3.5 rounded-full bg-emerald-400 shadow opacity-0 group-hover:opacity-100 transition-opacity"></div>
          </div>
        </div>
        <span class="w-9 md:w-10 shrink-0">{{ formatTime(duration) }}</span>
      </div>

      <!-- Buttons Row -->
      <div class="flex items-center justify-between">
        <div class="w-20 md:w-40">
          <button @click="cyclePlayMode"
            class="flex items-center gap-1 md:gap-1.5 px-2 md:px-3 py-1.5 rounded-xl hover:bg-white/8 text-gray-400 hover:text-white transition-all text-xs">
            <Shuffle v-if="playMode === 'random'" class="w-4 h-4 text-emerald-400" />
            <Repeat1 v-else-if="playMode === 'loop'" class="w-4 h-4 text-emerald-400" />
            <Repeat v-else class="w-4 h-4" />
            <span class="hidden sm:inline">{{ { sequence:'顺序播放', random:'随机播放', loop:'单曲循环' }[playMode] }}</span>
          </button>
        </div>

        <div class="flex items-center gap-5 md:gap-8">
          <button @click="$emit('prev')" class="text-gray-300 hover:text-white transition-colors p-1">
            <SkipBack class="w-5 h-5 md:w-6 md:h-6 fill-current" />
          </button>
          <button @click="$emit('toggle-play')"
            class="w-11 h-11 md:w-14 md:h-14 rounded-full bg-emerald-500 hover:bg-emerald-400 text-white flex items-center justify-center shadow-xl shadow-emerald-500/30 transition-all active:scale-95">
            <Pause v-if="isPlaying" class="w-5 h-5 md:w-7 md:h-7 fill-current" />
            <Play v-else class="w-5 h-5 md:w-7 md:h-7 fill-current translate-x-0.5" />
          </button>
          <button @click="$emit('next')" class="text-gray-300 hover:text-white transition-colors p-1">
            <SkipForward class="w-5 h-5 md:w-6 md:h-6 fill-current" />
          </button>
        </div>

        <div class="w-20 md:w-40 flex items-center justify-end gap-2 md:gap-3">
          <button @click="toggleMute" class="hidden sm:inline-flex text-gray-400 hover:text-white transition-colors">
            <VolumeX v-if="isMuted" class="w-4 h-4 text-red-400" />
            <Volume2 v-else class="w-4 h-4" />
          </button>
          <input type="range" min="0" max="1" step="0.01" v-model="volume"
            class="hidden sm:inline-block w-16 md:w-20 h-1 rounded-full appearance-none cursor-pointer accent-emerald-500 bg-white/10" />
          <div class="flex items-center gap-1 text-xs text-gray-400">
            <ListMusic class="w-3.5 h-3.5" />
            <span>{{ playlistCount }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import {
  ChevronDown, AlignCenter, ListMusic, Music, HardDrive, Download, Copy,
  Play, Pause, SkipBack, SkipForward, Shuffle, Repeat, Repeat1, Volume2, VolumeX
} from 'lucide-vue-next'

const props = defineProps({
  currentSong: Object,
  isPlaying: Boolean,
  currentTime: Number,
  duration: Number,
  lyricData: String,
  transData: String,
  playlistCount: { type: Number, default: 0 },
  playMode: { type: String, default: 'sequence' },
})

const emit = defineEmits(['collapse', 'toggle-play', 'prev', 'next', 'seek', 'volume-change', 'update:playMode'])

const lyricFontSize = ref(18)
const showTranslation = ref(true)
const mobileTab = ref('lyrics')
const volume = ref(0.8)
const isMuted = ref(false)
const prevVolume = ref(0.8)
const playModes = ['sequence', 'random', 'loop']
const lyricContainer = ref(null)
const progressBarRef = ref(null)
const lineRefs = ref([])

function setLineRef(el, idx) { if (el) lineRefs.value[idx] = el }

const parsedLyrics = computed(() => {
  if (!props.lyricData) return []
  let offsetSec = 0
  const offsetMatch = props.lyricData.match(/\[offset:\s*(-?\d+)\]/i)
  if (offsetMatch) {
    offsetSec = parseInt(offsetMatch[1], 10) / 1000
  }

  const transMap = new Map()
  if (props.transData) {
    for (const l of props.transData.split('\n')) {
      const tagRegex = /\[(\d{1,2}):(\d{2})(?:\.(\d{1,3}))?\]/g
      let match
      const text = l.replace(/\[\d{1,2}:\d{2}(?:\.\d{1,3})?\]/g, '').trim()
      while ((match = tagRegex.exec(l)) !== null) {
        const min = parseInt(match[1], 10)
        const sec = parseInt(match[2], 10)
        const msStr = (match[3] || '0').padEnd(3, '0').slice(0, 3)
        const totalSec = min * 60 + sec + parseInt(msStr, 10) / 1000 - offsetSec
        transMap.set(totalSec.toFixed(1), text)
      }
    }
  }

  const result = []
  for (const l of props.lyricData.split('\n')) {
    if (/^\[offset:/i.test(l) || /^\[(ti|ar|al|by|hash):/i.test(l)) continue
    const tagRegex = /\[(\d{1,2}):(\d{2})(?:\.(\d{1,3}))?\]/g
    let match
    const text = l.replace(/\[\d{1,2}:\d{2}(?:\.\d{1,3})?\]/g, '').trim()
    if (!text) continue
    while ((match = tagRegex.exec(l)) !== null) {
      const min = parseInt(match[1], 10)
      const sec = parseInt(match[2], 10)
      const msStr = (match[3] || '0').padEnd(3, '0').slice(0, 3)
      const totalSec = Math.max(0, min * 60 + sec + parseInt(msStr, 10) / 1000 - offsetSec)
      result.push({
        time: totalSec,
        text,
        trans: transMap.get(totalSec.toFixed(1)) || ''
      })
    }
  }
  return result.sort((a, b) => a.time - b.time)
})

const currentLineIndex = computed(() => {
  if (!parsedLyrics.value.length) return -1
  const t = props.currentTime
  for (let i = parsedLyrics.value.length - 1; i >= 0; i--) {
    if (t >= parsedLyrics.value[i].time) return i
  }
  return 0
})

watch(currentLineIndex, (idx) => {
  if (idx < 0 || !lyricContainer.value || !lineRefs.value[idx]) return
  nextTick(() => {
    const el = lineRefs.value[idx]
    const c = lyricContainer.value
    c.scrollTo({ top: el.offsetTop - c.clientHeight / 2 + el.clientHeight / 2, behavior: 'smooth' })
  })
})

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

function cyclePlayMode() {
  const nextIdx = (playModes.indexOf(props.playMode) + 1) % playModes.length
  emit('update:playMode', playModes[nextIdx])
}

function toggleMute() {
  if (isMuted.value) { isMuted.value = false; volume.value = prevVolume.value || 0.8 }
  else { prevVolume.value = volume.value; volume.value = 0; isMuted.value = true }
}

watch(volume, (val) => { emit('volume-change', parseFloat(val)); if (val > 0) isMuted.value = false })

function copyLyrics() {
  if (!props.lyricData) { alert('暂无歌词'); return }
  const text = parsedLyrics.value.map(l => l.text).join('\n')
  navigator.clipboard.writeText(text).then(() => alert('歌词已复制！')).catch(() => alert('复制失败'))
}
</script>
