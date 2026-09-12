<template>
  <div class="relative w-full h-full flex flex-col items-center select-none overflow-hidden py-4">
    <!-- Top info bar -->
    <div class="w-full flex items-center justify-between px-6 pb-2 text-xs text-slate-400 border-b border-white/5">
      <div class="flex items-center gap-2">
        <span class="font-medium text-slate-300">实时歌词</span>
        <span v-if="parsedLyrics.length" class="text-emerald-400 text-[11px]">• 词曲已同步 ({{ parsedLyrics.length }}行)</span>
        <span v-else class="text-amber-400 text-[11px]">• 纯音乐或暂无词</span>
      </div>
      <button 
        v-if="hasTranslation"
        @click="showTranslation = !showTranslation"
        :class="['px-2 py-0.5 rounded text-[11px] transition-colors', showTranslation ? 'bg-aurora-cyan/20 text-aurora-cyan font-bold' : 'bg-white/5 text-slate-400']"
      >
        译
      </button>
    </div>

    <!-- Lyrics Scroll Container -->
    <div 
      ref="containerRef" 
      class="w-full flex-1 overflow-y-auto overflow-x-hidden px-6 py-28 text-center space-y-6 scroll-smooth"
    >
      <div v-if="!parsedLyrics.length" class="h-full flex flex-col items-center justify-center text-slate-500">
        <p class="text-base font-medium">享受动人旋律...</p>
        <p class="text-xs text-slate-600 mt-1">沉浸在纯粹音乐空间</p>
      </div>

      <div 
        v-for="(line, idx) in parsedLyrics" 
        :key="idx"
        :ref="el => setLineRef(el, idx)"
        @click="seekTo(line.time)"
        :class="[
          'transition-all duration-300 cursor-pointer rounded-xl px-4 py-2 select-none inline-block max-w-xl',
          currentLineIndex === idx 
            ? 'lyric-active' 
            : 'text-slate-400/80 hover:text-slate-200 text-sm sm:text-base font-normal opacity-70 hover:opacity-100'
        ]"
      >
        <p class="leading-relaxed">{{ line.text }}</p>
        <p v-if="showTranslation && line.trans" class="text-xs mt-1 text-slate-400 font-normal">
          {{ line.trans }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'

const props = defineProps({
  lyricText: String,
  transText: String,
  currentTime: Number,
})

const emit = defineEmits(['seek'])

const showTranslation = ref(true)
const containerRef = ref(null)
const lineRefs = ref([])

function setLineRef(el, idx) {
  if (el) lineRefs.value[idx] = el
}

// Parse standard LRC format [00:12.34] lyric line
const parsedLyrics = computed(() => {
  if (!props.lyricText) return []
  const lines = props.lyricText.split('\n')
  const transMap = new Map()

  if (props.transText) {
    const tLines = props.transText.split('\n')
    for (const tl of tLines) {
      const match = tl.match(/\[(\d{2}):(\d{2})\.(\d{2,3})\](.*)/)
      if (match) {
        const sec = parseInt(match[1]) * 60 + parseFloat(match[2] + '.' + match[3])
        transMap.set(sec.toFixed(1), match[4].trim())
      }
    }
  }

  const result = []
  for (const l of lines) {
    const match = l.match(/\[(\d{2}):(\d{2})\.(\d{2,3})\](.*)/)
    if (match) {
      const sec = parseInt(match[1]) * 60 + parseFloat(match[2] + '.' + match[3])
      const text = match[4].trim()
      if (text) {
        result.push({
          time: sec,
          text,
          trans: transMap.get(sec.toFixed(1)) || '',
        })
      }
    }
  }
  return result.sort((a, b) => a.time - b.time)
})

const hasTranslation = computed(() => {
  return parsedLyrics.value.some(l => !!l.trans)
})

const currentLineIndex = computed(() => {
  if (!parsedLyrics.value.length) return -1
  const t = props.currentTime
  for (let i = parsedLyrics.value.length - 1; i >= 0; i--) {
    if (t >= parsedLyrics.value[i].time - 0.2) {
      return i
    }
  }
  return 0
})

watch(currentLineIndex, (newIdx) => {
  if (newIdx >= 0 && containerRef.value && lineRefs.value[newIdx]) {
    nextTick(() => {
      const lineEl = lineRefs.value[newIdx]
      const container = containerRef.value
      const targetScroll = lineEl.offsetTop - container.offsetTop - (container.clientHeight / 2) + (lineEl.clientHeight / 2)
      container.scrollTo({
        top: Math.max(0, targetScroll),
        behavior: 'smooth'
      })
    })
  }
})

function seekTo(sec) {
  emit('seek', sec)
}
</script>
