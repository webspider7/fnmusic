<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-all duration-200 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition-all duration-150 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div
        v-if="modelValue"
        class="fixed inset-0 z-[9999] flex items-center justify-center p-3 sm:p-4 bg-black/60 backdrop-blur-sm"
        style="z-index: 9999;"
        @click.self="close"
      >
        <div
          class="bg-white rounded-2xl shadow-2xl border border-gray-100 w-full max-w-md overflow-hidden flex flex-col max-h-[80vh] my-auto animate-in relative z-10"
          @keydown.enter.prevent="handleConfirm"
          @keydown.esc.prevent="close"
          tabindex="0"
          ref="modalRef"
        >
          <!-- 头部：歌曲基本信息 -->
          <div class="px-5 pt-4 pb-3.5 border-b border-gray-100 flex items-center justify-between shrink-0">
            <div class="flex items-center gap-3 min-w-0 pr-2">
              <div class="w-12 h-12 rounded-xl bg-gray-100 overflow-hidden shrink-0 shadow-xs flex items-center justify-center">
                <img
                  v-if="song?.cover"
                  :src="song.cover"
                  alt="cover"
                  referrerpolicy="no-referrer"
                  class="w-full h-full object-cover"
                  @error="song.cover = ''"
                />
                <Music2 v-else class="w-6 h-6 text-gray-400" />
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <h3 class="font-bold text-gray-900 text-sm truncate" :title="song?.name">
                    {{ song?.name || '未知歌曲' }}
                  </h3>
                  <span class="text-[10px] px-1.5 py-0.2 rounded-md bg-gray-100 text-gray-500 font-mono shrink-0">
                    {{ song?.source?.toUpperCase() || 'KW' }}
                  </span>
                </div>
                <p class="text-xs text-gray-400 truncate mt-0.5" :title="song?.singer">
                  {{ song?.singer || '未知歌手' }}{{ song?.album ? ' · ' + song.album : '' }}
                </p>
              </div>
            </div>
            <button
              @click="close"
              class="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors shrink-0 cursor-pointer"
            >
              <X class="w-4 h-4" />
            </button>
          </div>

          <!-- 主体：音质格式选择列表 -->
          <div class="p-5 overflow-y-auto flex-1 space-y-2.5">
            <div class="flex items-center justify-between mb-1">
              <span class="text-xs font-semibold text-gray-700 flex items-center gap-1.5">
                <Sparkles class="w-3.5 h-3.5 text-amber-500" />
                选择下载音质格式
              </span>
              <span class="text-[11px] text-gray-400">
                已自动匹配最高音质
              </span>
            </div>

            <!-- 加载状态 -->
            <div v-if="isLoadingQualities" class="py-8 flex flex-col items-center justify-center text-gray-400 gap-2">
              <Loader2 class="w-6 h-6 animate-spin text-emerald-500" />
              <span class="text-xs">正在获取音乐真实格式支持...</span>
            </div>

            <!-- 音质卡片列表 -->
            <div
              v-else
              v-for="(item, idx) in availableQualityList"
              :key="item.key"
              @click="selectedQuality = item.key"
              class="p-3 rounded-xl border transition-all cursor-pointer flex items-center justify-between gap-3 relative group"
              :class="[
                selectedQuality === item.key
                  ? 'border-emerald-500 bg-emerald-50/50 shadow-2xs'
                  : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50/60'
              ]"
            >
              <div class="flex items-start gap-3 min-w-0">
                <!-- 单选 Radio 按钮外观 -->
                <div
                  class="w-4 h-4 rounded-full mt-0.5 border flex items-center justify-center transition-colors shrink-0"
                  :class="[
                    selectedQuality === item.key
                      ? 'border-emerald-600 bg-emerald-600 text-white'
                      : 'border-gray-300 group-hover:border-gray-400'
                  ]"
                >
                  <Check v-if="selectedQuality === item.key" class="w-2.5 h-2.5 stroke-[3]" />
                </div>

                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="text-xs font-bold text-gray-900">
                      {{ item.title }}
                    </span>
                    <span
                      class="text-[9px] px-1.5 py-0.2 rounded-md font-bold font-mono border"
                      :class="item.badgeClass"
                    >
                      {{ item.format }}
                    </span>
                    <span
                      v-if="idx === 0"
                      class="text-[9px] px-1.5 py-0.2 rounded-md bg-amber-50 text-amber-700 font-medium border border-amber-200 shrink-0"
                    >
                      最高品质 (默认)
                    </span>
                  </div>
                  <p class="text-[11px] text-gray-500 mt-0.5 line-clamp-1">
                    {{ item.desc }}
                  </p>
                </div>
              </div>

              <!-- 选中的对勾指示器 -->
              <div
                v-if="selectedQuality === item.key"
                class="shrink-0 text-emerald-600"
              >
                <CheckCircle2 class="w-5 h-5 fill-emerald-100 text-emerald-600" />
              </div>
            </div>

            <!-- 若无有效解析出的格式，兜底显示标准品质 -->
            <div
              v-if="!isLoadingQualities && !availableQualityList.length"
              class="py-6 text-center text-xs text-gray-400"
            >
              未探测到该曲目的高阶无损格式，将默认以标准品质下载
            </div>
          </div>

          <!-- 底部动作条 -->
          <div class="px-5 py-3 bg-gray-50/90 border-t border-gray-100 flex items-center justify-between gap-3 shrink-0">
            <span class="text-[11px] text-gray-400 font-mono">
              格式将保存为对应真实后缀 (.flac / .mp3)
            </span>
            <div class="flex items-center gap-2">
              <button
                @click="close"
                class="px-3.5 py-1.5 rounded-xl text-xs text-gray-600 hover:bg-gray-200/60 transition-colors cursor-pointer"
              >
                取消
              </button>
              <button
                @click="handleConfirm"
                class="px-4 py-1.5 rounded-xl text-xs font-medium text-white bg-emerald-600 hover:bg-emerald-700 transition-colors shadow-xs flex items-center gap-1.5 cursor-pointer"
              >
                <Download class="w-3.5 h-3.5" />
                <span>立即下载</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { MusicAPI } from '../api/client'
import {
  X, Download, Check, CheckCircle2, Sparkles, Music2, Loader2
} from 'lucide-vue-next'

const props = defineProps({
  modelValue: Boolean,
  song: Object,
  activeSource: Object
})

const emit = defineEmits(['update:modelValue', 'confirm'])

const modalRef = ref(null)
const selectedQuality = ref('320k')
const rawQualities = ref([])
const isLoadingQualities = ref(false)

// 音质字典定义
const QUALITY_MAP = {
  flac24bit: {
    title: 'Hi-Res 母带无损',
    format: 'FLAC 24bit',
    desc: '24bit/96kHz+ 超高清无损录音室母带，动态范围更宽阔',
    badgeClass: 'bg-amber-50 text-amber-700 border-amber-200',
    order: 1
  },
  hires: {
    title: 'Hi-Res 母带无损',
    format: 'FLAC 24bit',
    desc: '24bit/96kHz+ 超高清无损录音室母带，动态范围更宽阔',
    badgeClass: 'bg-amber-50 text-amber-700 border-amber-200',
    order: 1
  },
  '24bit': {
    title: 'Hi-Res 母带无损',
    format: 'FLAC 24bit',
    desc: '24bit/96kHz+ 超高清无损录音室母带，动态范围更宽阔',
    badgeClass: 'bg-amber-50 text-amber-700 border-amber-200',
    order: 1
  },
  flac: {
    title: '超品无损',
    format: 'FLAC 16bit',
    desc: '16bit/44.1kHz CD级纯净无损音频，还原全部细节',
    badgeClass: 'bg-purple-50 text-purple-700 border-purple-200',
    order: 2
  },
  '320k': {
    title: '极高品质',
    format: 'MP3 320K',
    desc: '高比特率 MP3 格式，高频保留至 20kHz，音质出色',
    badgeClass: 'bg-emerald-50 text-emerald-700 border-emerald-200',
    order: 3
  },
  '192k': {
    title: '高清品质',
    format: 'MP3 192K',
    desc: '中高比特率格式，兼顾高保真与存储体积',
    badgeClass: 'bg-blue-50 text-blue-700 border-blue-200',
    order: 4
  },
  '128k': {
    title: '标准品质',
    format: 'MP3 128K',
    desc: '标准网络流媒体音频，文件体积小，快速下载',
    badgeClass: 'bg-gray-100 text-gray-700 border-gray-300',
    order: 5
  }
}

// 统一音质归一化键
function normalizeKey(q) {
  const lower = String(q || '').toLowerCase().trim()
  if (lower.includes('24') || lower.includes('hire')) return 'flac24bit'
  if (lower.includes('flac') || lower.includes('sq') || lower.includes('lossless')) return 'flac'
  if (lower.includes('320') || lower.includes('hq') || lower.includes('exhigh')) return '320k'
  if (lower.includes('192')) return '192k'
  if (lower.includes('128') || lower.includes('standard')) return '128k'
  return lower || '128k'
}

// 计算排序后的可用音质选项列表
const availableQualityList = computed(() => {
  const seen = new Set()
  const list = []

  for (const raw of rawQualities.value) {
    const key = normalizeKey(raw)
    if (seen.has(key)) continue
    seen.add(key)

    const meta = QUALITY_MAP[key] || {
      title: key.toUpperCase(),
      format: key.toUpperCase(),
      desc: '对应音频格式',
      badgeClass: 'bg-gray-100 text-gray-600 border-gray-200',
      order: 10
    }

    list.push({
      key,
      ...meta
    })
  }

  // 确保按音质由高到低严格排序
  list.sort((a, b) => a.order - b.order)
  return list
})

// 当弹窗打开或传入新歌曲时，拉取或解析真实支持的音质
watch(
  () => [props.modelValue, props.song],
  async ([isOpen, song]) => {
    if (!isOpen || !song) return

    nextTick(() => {
      if (modalRef.value) modalRef.value.focus()
    })

    // 1. 如果歌曲本身已有后端返回的真实 qualitys
    if (Array.isArray(song.qualitys) && song.qualitys.length > 0) {
      rawQualities.value = [...song.qualitys]
      applyDefaultHighest()
      return
    }

    // 2. 否则动态向后端请求探测真实格式
    isLoadingQualities.value = true
    try {
      const res = await MusicAPI.getQualities(
        song.source,
        song.id,
        song.songmid,
        song.hash
      )
      if (res?.code === 200 && Array.isArray(res.qualities) && res.qualities.length > 0) {
        rawQualities.value = res.qualities
      } else {
        // 保守兜底
        rawQualities.value = ['320k', '128k']
      }
    } catch (_) {
      rawQualities.value = ['320k', '128k']
    } finally {
      isLoadingQualities.value = false
      applyDefaultHighest()
    }
  },
  { immediate: true }
)

// 默认高亮选择最高品质
function applyDefaultHighest() {
  const list = availableQualityList.value
  if (list.length > 0) {
    // 列表已按由高到低排序，第一项即为当前曲目真实拥有的最高品质
    selectedQuality.value = list[0].key
  } else {
    selectedQuality.value = '320k'
  }
}

function close() {
  emit('update:modelValue', false)
}

function handleConfirm() {
  emit('confirm', props.song, selectedQuality.value)
  close()
}
</script>
