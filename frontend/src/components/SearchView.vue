<template>
  <div class="h-full flex flex-col overflow-hidden select-none bg-gray-50">
    <!-- ── 顶部搜索与导航条 ── -->
    <div class="px-3 sm:px-6 pt-3 sm:pt-4 pb-3 bg-white border-b border-gray-100 shrink-0">
      <div class="max-w-4xl mx-auto space-y-2.5">
        <!-- 搜索输入框 -->
        <div class="relative">
          <Search class="absolute left-3.5 top-2.5 w-4 h-4 text-gray-400" />
          <input
            v-model="query"
            @keyup.enter="handleSearch"
            type="text"
            placeholder="搜索歌曲、歌手、专辑..."
            class="w-full pl-10 pr-24 py-2 rounded-xl bg-gray-50 border border-gray-200 text-gray-800 text-xs sm:text-sm placeholder:text-gray-400 focus:outline-none focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400/20 transition-all"
          />
          <div class="absolute right-1 top-1 flex items-center gap-1">
            <button
              v-if="query || currentMode !== 'discover'"
              @click="resetToDiscover"
              class="p-1 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
              title="清空并返回发现"
            >
              <X class="w-3.5 h-3.5" />
            </button>
            <button
              @click="handleSearch"
              :disabled="isSearching"
              class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white font-semibold text-xs shadow-sm disabled:opacity-50 transition-all"
            >
              {{ isSearching ? '搜索中...' : '搜索' }}
            </button>
          </div>
        </div>

        <!-- 热门搜索词快捷标签 -->
        <div class="flex items-center gap-2 text-xs">
          <span class="text-gray-400 shrink-0 text-[11px]">热门:</span>
          <div class="flex flex-wrap gap-1.5 overflow-hidden max-h-6">
            <button
              v-for="tag in hotTags" :key="tag"
              @click="quickSearch(tag)"
              class="px-2 py-0.5 rounded-md bg-gray-100 hover:bg-emerald-50 hover:text-emerald-600 text-gray-500 transition-colors text-[11px]"
            >{{ tag }}</button>
          </div>
        </div>

        <!-- 发现模式切换选项卡 (未搜索时显示) -->
        <div v-if="currentMode === 'discover'" class="flex flex-wrap items-center gap-2 pt-1">
          <!-- 动态音源渠道选择器 -->
          <div v-if="availablePlatforms.length > 0" class="flex items-center bg-gray-100 p-0.5 rounded-lg text-xs shrink-0">
            <button
              v-for="p in availablePlatforms" :key="p.id"
              @click="switchChartPlatform(p.id)"
              :class="[
                'px-2.5 py-1 rounded-md font-medium transition-all flex items-center gap-1 text-[11px]',
                selectedChartPlatform === p.id ? 'bg-white text-emerald-600 shadow-xs font-semibold' : 'text-gray-500 hover:text-gray-700'
              ]"
            >
              <span>{{ p.icon }}</span>
              <span>{{ p.name }}</span>
            </button>
          </div>

          <div class="flex items-center bg-gray-100 p-0.5 rounded-lg text-xs shrink-0">
            <button
              @click="switchDiscoverTab('charts')"
              :class="[
                'px-3 py-1 rounded-md font-medium transition-all flex items-center gap-1 text-[11px]',
                discoverTab === 'charts' ? 'bg-white text-emerald-600 shadow-xs font-semibold' : 'text-gray-500 hover:text-gray-700'
              ]"
            >
              <span>🔥</span>
              <span>排行榜单</span>
            </button>
            <button
              @click="switchDiscoverTab('playlists')"
              :class="[
                'px-3 py-1 rounded-md font-medium transition-all flex items-center gap-1 text-[11px]',
                discoverTab === 'playlists' ? 'bg-white text-emerald-600 shadow-xs font-semibold' : 'text-gray-500 hover:text-gray-700'
              ]"
            >
              <span>🎧</span>
              <span>精选歌单</span>
            </button>
          </div>

          <!-- 歌单分类标签 (网易平台支持分类) -->
          <div v-if="discoverTab === 'playlists' && selectedChartPlatform === 'wy'" class="flex items-center gap-1 overflow-x-auto pb-0.5 text-xs">
            <button
              v-for="cat in playlistCategories" :key="cat"
              @click="selectPlaylistCategory(cat)"
              :class="[
                'px-2 py-0.5 rounded-full text-[11px] whitespace-nowrap transition-all',
                selectedCategory === cat ? 'bg-emerald-500 text-white font-semibold' : 'bg-gray-100 text-gray-500 hover:bg-gray-200'
              ]"
            >
              {{ cat }}
            </button>
          </div>
        </div>

        <!-- 搜索状态下的平台切换标签 -->
        <div v-if="currentMode === 'search'" class="flex items-center gap-1.5 overflow-x-auto pb-0.5">
          <button
            v-for="p in platforms" :key="p.id"
            @click="switchPlatform(p.id)"
            :class="[
              'px-2.5 sm:px-3 py-1 rounded-lg text-xs font-medium transition-all whitespace-nowrap flex items-center gap-1',
              selectedPlatform === p.id ? 'bg-emerald-500 text-white shadow-sm' : 'bg-gray-100 text-gray-500 hover:bg-gray-200'
            ]"
          >
            <span>{{ p.icon }}</span><span>{{ p.name }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- ── 内容展示主区域 ── -->
    <div class="flex-1 overflow-y-auto">

      <!-- 0. 未导入音源时的合规纯净空状态 -->
      <div v-if="currentMode === 'discover' && !hasActiveSource" class="flex flex-col items-center justify-center py-20 text-center px-4 max-w-md mx-auto">
        <div class="w-16 h-16 rounded-2xl bg-emerald-50 text-emerald-600 flex items-center justify-center mb-4 shadow-sm">
          <Radio class="w-8 h-8" />
        </div>
        <h3 class="text-base font-bold text-gray-800 mb-2">未导入第三方音源</h3>
        <p class="text-xs text-gray-500 leading-relaxed mb-6">
          本播放器遵循纯净播放器规范，不内置任何在线曲库或固定平台榜单。<br><br>
          请前往「音源管理」导入您的音源脚本，系统将根据音源声明支持的平台（如 kg、wy、tx、kw 等）动态呈现对应官方权威榜单与精选歌单。
        </p>
        <button
          @click="emit('navigate', 'sources')"
          class="px-5 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-semibold shadow-sm transition-all flex items-center gap-1.5 cursor-pointer"
        >
          <span>➕ 前往「音源管理」导入音源</span>
        </button>
      </div>

      <!-- 1. 发现大厅 - 排行榜单网格 (紧凑精致卡片排版) -->
      <div v-else-if="currentMode === 'discover' && discoverTab === 'charts'" class="p-3 sm:p-5 max-w-5xl mx-auto">
        <div v-if="isLoadingCharts" class="flex flex-col items-center justify-center py-20 text-gray-400">
          <Loader2 class="w-8 h-8 animate-spin text-emerald-500 mb-2" />
          <p class="text-xs">正在载入权威榜单...</p>
        </div>
        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          <div
            v-for="chart in toplists" :key="chart.id"
            @click="openChartDetail(chart)"
            class="bg-white rounded-xl border border-gray-100 p-2.5 hover:border-emerald-300 hover:shadow-sm transition-all cursor-pointer flex items-center gap-3 group"
          >
            <!-- 固定 76x76px 尺寸封面，杜绝撑大 -->
            <div
              class="relative rounded-lg overflow-hidden bg-gray-100 shrink-0 shadow-2xs"
              style="width: 76px; height: 76px; min-width: 76px; min-height: 76px; max-width: 76px; max-height: 76px;"
            >
              <img :src="chart.cover" alt="cover" referrerpolicy="no-referrer" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" />
              <div class="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
                <div class="w-7 h-7 rounded-full bg-emerald-500 text-white flex items-center justify-center shadow-md">
                  <Play class="w-3.5 h-3.5 ml-0.5 fill-current" />
                </div>
              </div>
              <span v-if="chart.update_frequency" class="absolute bottom-1 right-1 text-[8px] px-1 py-0.2 rounded bg-black/60 text-white backdrop-blur">
                {{ chart.update_frequency }}
              </span>
            </div>

            <!-- 右侧歌曲预览 -->
            <div class="flex-1 min-w-0 pr-1">
              <h4 class="text-xs sm:text-sm font-bold text-gray-800 group-hover:text-emerald-600 transition-colors truncate mb-1">
                {{ chart.name }}
              </h4>
              <div class="space-y-0.5">
                <p v-for="(t, idx) in chart.tracks.slice(0, 3)" :key="idx" class="text-[11px] text-gray-500 truncate leading-tight">
                  <span class="text-gray-400 font-mono text-[10px] mr-1">{{ idx + 1 }}.</span>
                  {{ t }}
                </p>
                <p v-if="!chart.tracks.length" class="text-[10px] text-gray-300 italic">点击查看完整歌曲</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 2. 发现大厅 - 精选歌单网格 -->
      <div v-else-if="currentMode === 'discover' && discoverTab === 'playlists'" class="p-3 sm:p-5 max-w-5xl mx-auto">
        <div v-if="isLoadingPlaylists" class="flex flex-col items-center justify-center py-20 text-gray-400">
          <Loader2 class="w-8 h-8 animate-spin text-emerald-500 mb-2" />
          <p class="text-xs">正在载入精选歌单...</p>
        </div>
        <div v-else class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3 sm:gap-3.5">
          <div
            v-for="pl in playlists" :key="pl.id"
            @click="openPlaylistDetail(pl)"
            class="bg-white rounded-xl border border-gray-100 p-2 hover:border-emerald-300 hover:shadow-sm transition-all cursor-pointer group flex flex-col"
          >
            <div class="relative aspect-square rounded-lg overflow-hidden bg-gray-100 mb-2">
              <img :src="pl.cover" alt="cover" referrerpolicy="no-referrer" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" />
              <div class="absolute inset-0 bg-black/20 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
                <div class="w-8 h-8 rounded-full bg-emerald-500 text-white flex items-center justify-center shadow-lg">
                  <Play class="w-4 h-4 ml-0.5 fill-current" />
                </div>
              </div>
              <span class="absolute top-1.5 right-1.5 text-[9px] px-1.5 py-0.5 rounded-full bg-black/50 text-white backdrop-blur flex items-center gap-0.5">
                🔥 {{ formatPlayCount(pl.play_count) }}
              </span>
            </div>
            <h4 class="text-xs font-semibold text-gray-800 line-clamp-2 leading-snug group-hover:text-emerald-600 transition-colors mb-1">
              {{ pl.title }}
            </h4>
            <p class="text-[10px] text-gray-400 truncate mt-auto">
              {{ pl.track_count ? pl.track_count + ' 首 · ' : '' }}{{ pl.creator }}
            </p>
          </div>
        </div>
      </div>

      <!-- 3. 歌曲列表视图 (搜索结果 / 榜单详情 / 歌单详情 公用) -->
      <div v-else class="max-w-5xl mx-auto p-3 sm:p-5 space-y-3">
        <!-- 头部导航与批量操作工具栏 -->
        <div class="bg-white rounded-xl border border-gray-100 p-3.5 shadow-xs">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-2.5">
            <div class="flex items-center gap-2.5 min-w-0">
              <button
                @click="resetToDiscover"
                class="p-1.5 rounded-lg border border-gray-200 text-gray-500 hover:bg-gray-100 transition-colors shrink-0"
                title="返回发现"
              >
                <ArrowLeft class="w-4 h-4" />
              </button>
              <div
                class="rounded-lg overflow-hidden bg-gray-100 shrink-0 flex items-center justify-center shadow-2xs"
                style="width: 44px; height: 44px; min-width: 44px; min-height: 44px;"
              >
                <img v-if="currentDetailMeta?.cover" :src="currentDetailMeta.cover" alt="cover" referrerpolicy="no-referrer" class="w-full h-full object-cover" />
                <span v-else class="text-lg">🎵</span>
              </div>
              <div class="min-w-0">
                <h3 class="text-sm font-bold text-gray-800 flex items-center gap-2 truncate">
                  <span class="truncate">{{ currentDetailMeta?.title || (query ? `搜索：“${query}”` : '歌曲列表') }}</span>
                  <span class="text-xs text-gray-400 font-normal shrink-0">({{ activeSongList.length }} 首)</span>
                </h3>
                <p class="text-xs text-gray-400 truncate max-w-md">
                  {{ currentDetailMeta?.desc || (currentMode === 'search' ? '跨平台聚合搜索结果' : '极光音乐播放列表') }}
                </p>
              </div>
            </div>

            <!-- 控制按钮群 -->
            <div class="flex items-center gap-2 flex-wrap shrink-0">
              <!-- 一键播放全部 -->
              <button
                @click="playAllSongs"
                :disabled="!activeSongList.length"
                class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-semibold flex items-center gap-1.5 shadow-sm transition-all disabled:opacity-50"
              >
                <Play class="w-3.5 h-3.5 fill-current" />
                <span>播放全部</span>
              </button>

              <!-- 播放模式切换 (顺序、随机、循环) -->
              <button
                @click="cyclePlayMode"
                class="px-2.5 py-1.5 rounded-lg bg-gray-100 hover:bg-gray-200 text-gray-700 text-xs font-medium flex items-center gap-1.5 transition-all"
                :title="'当前模式：' + playModeLabel"
              >
                <component :is="playModeIcon" class="w-3.5 h-3.5 text-emerald-600" />
                <span>{{ playModeLabel }}</span>
              </button>

              <!-- 一键批量下载到 NAS -->
              <button
                @click="batchDownloadAll"
                :disabled="!activeSongList.length"
                class="px-3 py-1.5 rounded-lg border transition-all text-xs font-semibold flex items-center gap-1.5 shadow-xs bg-emerald-50 hover:bg-emerald-100 border-emerald-200 text-emerald-700 cursor-pointer"
                title="一键下载当前全部歌曲到 NAS"
              >
                <HardDriveDownload class="w-3.5 h-3.5 shrink-0" />
                <span>一键下载全部 ({{ activeSongList.length }})</span>
              </button>

              <!-- 设置下载目录 -->
              <button
                @click="downloadManager.openDirModal()"
                class="px-2 py-1.5 rounded-lg border border-gray-200 hover:bg-gray-100 text-gray-600 text-xs flex items-center gap-1 transition-all cursor-pointer"
                :title="'当前保存目录：' + downloadManager.currentDownloadDir.value"
              >
                <Folder class="w-3.5 h-3.5 text-amber-500" />
                <span class="max-w-[90px] truncate hidden sm:inline">{{ downloadManager.currentDownloadDir.value }}</span>
                <span class="sm:hidden">设置目录</span>
              </button>
            </div>
          </div>
        </div>

        <!-- 歌曲列表主体 -->
        <div v-if="isLoadingDetail || isSearching" class="flex flex-col items-center justify-center py-20 text-gray-400">
          <Loader2 class="w-8 h-8 animate-spin text-emerald-500 mb-2" />
          <p class="text-xs">{{ isSearching ? '正在检索歌曲...' : '正在载入曲目列表...' }}</p>
        </div>

        <div v-else-if="!activeSongList.length" class="flex flex-col items-center justify-center py-20 text-gray-400 bg-white rounded-xl border border-gray-100">
          <Music2 class="w-10 h-10 text-gray-300 mb-2" />
          <p class="text-sm font-semibold text-gray-600">未找到相关歌曲</p>
          <p class="text-xs text-gray-400 mt-1">请尝试更换关键词或在「音源管理」载入更多音源</p>
        </div>

        <div v-else class="bg-white rounded-xl border border-gray-100 divide-y divide-gray-100 overflow-hidden shadow-2xs">
          <div
            v-for="(song, idx) in activeSongList" :key="song.id || idx"
            @click="playSong(song, idx)"
            class="flex items-center gap-2.5 sm:gap-3 px-3 sm:px-4 py-2.5 hover:bg-emerald-50/40 cursor-pointer transition-colors group"
          >
            <!-- 序号 / 播放小图标 -->
            <span class="w-5 sm:w-6 text-center text-xs text-gray-300 font-mono group-hover:hidden shrink-0">{{ idx + 1 }}</span>
            <div class="w-5 sm:w-6 hidden group-hover:flex items-center justify-center shrink-0">
              <Play class="w-3.5 h-3.5 text-emerald-500 fill-current" />
            </div>

            <!-- 封面缩略图 -->
            <div class="w-9 h-9 rounded-lg overflow-hidden bg-gray-100 shrink-0 flex items-center justify-center shadow-2xs">
              <img v-if="song.cover" :src="song.cover" alt="cover" referrerpolicy="no-referrer" class="w-full h-full object-cover" @error="song.cover = ''" />
              <span v-else class="text-base">🎵</span>
            </div>

            <!-- 歌名与歌手 -->
            <div class="flex-1 min-w-0">
              <p class="text-xs sm:text-sm font-medium text-gray-800 truncate group-hover:text-emerald-600 transition-colors">
                {{ song.name }}
              </p>
              <p class="text-[11px] text-gray-400 truncate">
                {{ song.singer }}{{ song.album ? ' · ' + song.album : '' }}
              </p>
            </div>

            <!-- 渠道标识 -->
            <span class="text-[10px] px-1.5 py-0.5 rounded-full bg-gray-100 text-gray-500 font-mono shrink-0">
              {{ song.source || 'wy' }}
            </span>

            <!-- 时长 -->
            <span class="text-xs text-gray-400 font-mono shrink-0 w-11 text-right">
              {{ formatDuration(song.interval || song.duration) }}
            </span>

            <!-- 单曲下载到 NAS 按钮 -->
            <div class="shrink-0" @click.stop>
              <!-- 已下载 -->
              <span
                v-if="downloadManager.isSongDownloaded(song)"
                class="px-2 py-0.5 rounded-lg bg-emerald-50 text-emerald-600 text-[11px] font-medium flex items-center gap-1 border border-emerald-200"
                title="已保存至 NAS 存储目录"
              >
                <Check class="w-3.5 h-3.5" />
                <span class="hidden sm:inline">已在NAS</span>
              </span>

              <!-- 下载中 (点击可打开下载任务抽屉) -->
              <span
                v-else-if="downloadManager.isSongDownloading(song)"
                @click="downloadManager.open()"
                class="px-2 py-0.5 rounded-lg bg-emerald-50/60 text-emerald-600 text-[11px] flex items-center gap-1 cursor-pointer hover:bg-emerald-100 transition-colors"
                title="点击查看下载任务进度"
              >
                <Loader2 class="w-3.5 h-3.5 animate-spin text-emerald-500" />
                <span class="hidden sm:inline">下载中</span>
              </span>

              <!-- 未下载，点击下载 (自动加入全局后台队列并滑出下载面板) -->
              <button
                v-else
                @click="downloadSingleSong(song)"
                class="p-1.5 rounded-lg hover:bg-emerald-50 text-gray-400 hover:text-emerald-600 transition-colors cursor-pointer"
                title="下载歌曲到 NAS"
              >
                <Download class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ── 轻量下载操作提示 Toast ── -->
    <Teleport to="body">
      <Transition
        enter-active-class="transition-all duration-200 ease-out"
        enter-from-class="opacity-0 translate-y-2 scale-95"
        enter-to-class="opacity-100 translate-y-0 scale-100"
        leave-active-class="transition-all duration-150 ease-in"
        leave-from-class="opacity-100 translate-y-0 scale-100"
        leave-to-class="opacity-0 translate-y-2 scale-95"
      >
        <div
          v-if="toastMessage"
          class="fixed bottom-22 left-1/2 -translate-x-1/2 z-60 bg-gray-900/90 text-white text-xs px-4 py-2 rounded-full shadow-lg backdrop-blur-xs flex items-center gap-2 pointer-events-none"
        >
          <Check class="w-3.5 h-3.5 text-emerald-400" />
          <span>{{ toastMessage }}</span>
        </div>
      </Transition>
    </Teleport>

    <!-- ── 下载音质格式选择模态框 ── -->
    <DownloadQualityModal
      v-model="isQualityModalOpen"
      :song="currentDownloadSong"
      :active-source="activeSource"
      @confirm="handleConfirmDownloadQuality"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { SearchAPI, ChartsAPI } from '../api/client'
import { lxRuntime } from '../engine/lx-runtime'
import { downloadManager } from '../services/downloadManager'
import DownloadQualityModal from './DownloadQualityModal.vue'
import {
  Search, Play, Loader2, Music2, Download, HardDriveDownload,
  Folder, ArrowLeft, Shuffle, Repeat, Repeat1, Check, X, Radio
} from 'lucide-vue-next'

const props = defineProps({
  activeSource: Object,
  playMode: {
    type: String,
    default: 'sequence'
  }
})

const emit = defineEmits(['play', 'play-all', 'update:play-mode', 'navigate'])

// 状态控制
const currentMode = ref('discover') // 'discover' | 'chartDetail' | 'playlistDetail' | 'search'
const discoverTab = ref('charts')   // 'charts' | 'playlists'
const selectedCategory = ref('全部')
const query = ref('')
const selectedPlatform = ref('all')

// 数据列表
const toplists = ref([])
const playlists = ref([])
const searchResults = ref([])
const activeSongList = ref([])
const currentDetailMeta = ref(null)

// 状态标记
const isLoadingCharts = ref(false)
const isLoadingPlaylists = ref(false)
const isLoadingDetail = ref(false)
const isSearching = ref(false)

const hotTags = ['周杰伦', '刀郎', '邓紫棋', '林俊杰', '陈奕迅', '华语经典', '赛博朋克电音', '新歌推荐']
const playlistCategories = ['全部', '华语', '流行', '摇滚', '民谣', '经典怀旧', '轻音乐', 'ACG', '车载', '电音', '欧美', '粤语']

const platformNames = {
  wy: 'wy',
  kg: 'kg',
  tx: 'tx',
  kw: 'kw',
}

const platformIcons = {
  all: '🌐',
  wy: '🔴',
  tx: '🟢',
  kg: '🔵',
  kw: '🟡',
}

const defaultPlatforms = [
  { id: 'kw', name: 'kw', icon: '🟡' },
  { id: 'kg', name: 'kg', icon: '🔵' },
  { id: 'tx', name: 'tx', icon: '🟢' },
  { id: 'wy', name: 'wy', icon: '🔴' },
]

const hasActiveSource = computed(() => {
  if (props.activeSource?.id) return true
  try {
    const cached = localStorage.getItem('fn_active_source_meta')
    if (cached && JSON.parse(cached)?.id) return true
  } catch (_) {}
  return false
})

const STANDARD_PLATFORMS = ['wy', 'tx', 'kw', 'kg']

const availablePlatforms = computed(() => {
  if (props.activeSource?.id) {
    const supported = lxRuntime.getSupportedSources(props.activeSource.id)
    const list = supported
      .filter(id => STANDARD_PLATFORMS.includes(id))
      .map(id => ({
        id,
        name: platformNames[id] || id,
        icon: platformIcons[id] || '🎵'
      }))
    if (list.length > 0) return list
  }
  return defaultPlatforms
})

const selectedChartPlatform = ref('kw')

const platforms = computed(() => {
  const list = [{ id: 'all', name: '全网聚合', icon: '🌐' }]
  if (props.activeSource?.id) {
    const supported = lxRuntime.getSupportedSources(props.activeSource.id)
    const filtered = supported.filter(id => STANDARD_PLATFORMS.includes(id))
    if (filtered.length > 0) {
      filtered.forEach(id => {
        list.push({
          id,
          name: platformNames[id] || id,
          icon: platformIcons[id] || '🎵'
        })
      })
      return list
    }
  }
  defaultPlatforms.forEach(p => list.push(p))
  return list
})

function switchChartPlatform(pid) {
  selectedChartPlatform.value = pid
  toplists.value = []
  playlists.value = []
  if (discoverTab.value === 'charts') {
    loadToplists(pid)
  } else {
    loadPlaylists(selectedCategory.value, pid)
  }
}

function switchDiscoverTab(tab) {
  discoverTab.value = tab
  if (tab === 'charts') {
    loadToplists(selectedChartPlatform.value)
  } else {
    loadPlaylists(selectedCategory.value, selectedChartPlatform.value)
  }
}

watch(() => props.activeSource?.id, (newId) => {
  if (!newId) return
  const list = lxRuntime.getSupportedSources(newId).filter(id => STANDARD_PLATFORMS.includes(id))
  if (list.length > 0 && !list.includes(selectedChartPlatform.value)) {
    selectedChartPlatform.value = list[0]
  }
  if (discoverTab.value === 'charts') {
    if (toplists.value.length === 0) {
      loadToplists(selectedChartPlatform.value)
    }
  } else {
    if (playlists.value.length === 0) {
      loadPlaylists(selectedCategory.value, selectedChartPlatform.value)
    }
  }
}, { immediate: true })

// 播放模式控制
const playModeIcon = computed(() => {
  if (props.playMode === 'random') return Shuffle
  if (props.playMode === 'loop') return Repeat1
  return Repeat
})

const playModeLabel = computed(() => {
  if (props.playMode === 'random') return '随机播放'
  if (props.playMode === 'loop') return '单曲循环'
  return '顺序播放'
})

function cyclePlayMode() {
  const modes = ['sequence', 'random', 'loop']
  const next = modes[(modes.indexOf(props.playMode) + 1) % modes.length]
  emit('update:play-mode', next)
}

function formatDuration(sec) {
  if (!sec) return '—'
  const m = Math.floor(sec / 60).toString().padStart(2, '0')
  const s = Math.floor(sec % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

function formatPlayCount(num) {
  if (!num) return '0'
  if (num > 100000000) return (num / 100000000).toFixed(1) + '亿'
  if (num > 10000) return (num / 10000).toFixed(1) + '万'
  return num.toString()
}

// ── 榜单与歌单加载 ──
async function loadToplists(platform = selectedChartPlatform.value) {
  isLoadingCharts.value = true
  toplists.value = []
  try {
    const res = await ChartsAPI.toplists(platform)
    if (res.code === 200) {
      toplists.value = res.data || []
    }
  } catch (e) {
    console.warn('Failed to load toplists:', e)
    toplists.value = []
  } finally {
    isLoadingCharts.value = false
  }
}

async function loadPlaylists(cat = selectedCategory.value, platform = selectedChartPlatform.value) {
  isLoadingPlaylists.value = true
  playlists.value = []
  try {
    const res = await ChartsAPI.playlists(cat, 1, 24, platform)
    if (res.code === 200) {
      playlists.value = res.data || []
    }
  } catch (e) {
    console.warn('Failed to load playlists:', e)
    playlists.value = []
  } finally {
    isLoadingPlaylists.value = false
  }
}

function selectPlaylistCategory(cat) {
  selectedCategory.value = cat
  loadPlaylists(cat, selectedChartPlatform.value)
}

async function openChartDetail(chart) {
  currentMode.value = 'chartDetail'
  currentDetailMeta.value = {
    title: chart.name,
    desc: chart.update_frequency ? `更新频率：${chart.update_frequency}` : '官方排行榜单',
    cover: chart.cover,
  }
  isLoadingDetail.value = true
  activeSongList.value = []
  try {
    const res = await ChartsAPI.chartDetail(chart.id, chart.source || selectedChartPlatform.value, chart.name)
    if (res.code === 200 && res.data) {
      activeSongList.value = res.data.songs || []
      checkSongsDownloaded(activeSongList.value)
    }
  } catch (e) {
    alert('加载榜单歌曲失败: ' + e.message)
  } finally {
    isLoadingDetail.value = false
  }
}

async function openPlaylistDetail(pl) {
  currentMode.value = 'playlistDetail'
  currentDetailMeta.value = {
    title: pl.title,
    desc: pl.creator ? `创建者：${pl.creator}` : (pl.description || '精选歌单'),
    cover: pl.cover,
  }
  isLoadingDetail.value = true
  activeSongList.value = []
  try {
    const res = await ChartsAPI.playlistDetail(pl.id, pl.source || selectedChartPlatform.value, pl.title)
    if (res.code === 200 && res.data) {
      activeSongList.value = res.data.songs || []
      checkSongsDownloaded(activeSongList.value)
    }
  } catch (e) {
    alert('加载歌单歌曲失败: ' + e.message)
  } finally {
    isLoadingDetail.value = false
  }
}

// ── 搜索处理 ──
async function handleSearch() {
  const q = query.value.trim()
  if (!q) return
  if (!hasActiveSource.value) {
    alert('【未导入第三方音源】\n\n本播放器为纯本地播放器容器，不内置在线曲库与检索。\n请先前往「音源管理」导入第三方音源脚本后再发起搜索。')
    return
  }
  currentMode.value = 'search'
  currentDetailMeta.value = {
    title: `搜索：“${q}”`,
    desc: '跨平台聚合检索',
    cover: '',
  }
  isSearching.value = true
  activeSongList.value = []
  try {
    const res = await SearchAPI.search(q, selectedPlatform.value, 1)
    if (res.code === 200) {
      searchResults.value = res.data.list || []
      activeSongList.value = searchResults.value
      checkSongsDownloaded(activeSongList.value)
    }
  } catch (e) {
    alert('搜索出错: ' + e.message)
  } finally {
    isSearching.value = false
  }
}

function quickSearch(tag) {
  query.value = tag
  handleSearch()
}

function switchPlatform(p) {
  selectedPlatform.value = p
  if (query.value.trim()) {
    handleSearch()
  }
}

function resetToDiscover() {
  query.value = ''
  currentMode.value = 'discover'
  activeSongList.value = []
  currentDetailMeta.value = null
}

// ── 播放触发 (关键：点击单曲带上完整歌单上下文，自动顺序播放本歌单/榜单所有歌曲) ──
function playSong(song, idx) {
  emit('play', {
    song,
    playlist: [...activeSongList.value],
    index: idx !== undefined ? idx : activeSongList.value.findIndex(s => s.id === song.id)
  })
}

function playAllSongs() {
  if (!activeSongList.value.length) return
  emit('play-all', {
    songs: [...activeSongList.value],
    startIndex: 0
  })
}

// ── 下载与状态同步 ──
const toastMessage = ref('')
let toastTimer = null

function showToast(msg) {
  toastMessage.value = msg
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toastMessage.value = ''
  }, 2500)
}

const isQualityModalOpen = ref(false)
const currentDownloadSong = ref(null)

function checkSongsDownloaded(songs) {
  downloadManager.checkSongsDownloaded(songs)
}

function downloadSingleSong(song) {
  if (!props.activeSource?.id && song.source !== 'nas') {
    alert('【未导入第三方音源】\n\n无法下载网络歌曲，请先在「音源管理」导入第三方音源脚本。')
    return
  }
  currentDownloadSong.value = song
  isQualityModalOpen.value = true
}

function handleConfirmDownloadQuality(song, quality) {
  downloadManager.addSong(song, props.activeSource?.id, { autoOpen: false, quality })
  const qText = quality ? quality.toUpperCase() : '默认'
  showToast(`已将《${song.name}》(${qText})加入后台下载队列`)
}

function batchDownloadAll() {
  const list = activeSongList.value
  if (!list.length) return

  if (!hasActiveSource.value && list.some(s => s.source !== 'nas')) {
    alert('【未导入第三方音源】\n\n无法下载网络歌曲，请先在「音源管理」导入第三方音源脚本。')
    return
  }

  downloadManager.addBatch(list, props.activeSource?.id, false)
  showToast(`已添加 ${list.length} 首歌曲至后台下载队列 (默认最高品质)`)
}

onMounted(async () => {
  loadToplists(selectedChartPlatform.value)
  loadPlaylists('全部', selectedChartPlatform.value)
})
</script>
