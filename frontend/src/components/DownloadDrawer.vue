<template>
  <div>
    <!-- ── 右侧滑出式下载任务抽屉 ── -->
    <Teleport to="body">
      <!-- 遮罩背景 -->
      <Transition
        enter-active-class="transition-opacity duration-300 ease-out"
        enter-from-class="opacity-0"
        enter-to-class="opacity-100"
        leave-active-class="transition-opacity duration-200 ease-in"
        leave-from-class="opacity-100"
        leave-to-class="opacity-0"
      >
        <div
          v-if="downloadManager.isOpen.value"
          @click="downloadManager.close()"
          class="fixed inset-0 z-50 bg-black/40 backdrop-blur-xs"
        ></div>
      </Transition>

      <!-- 抽屉主体 -->
      <Transition
        enter-active-class="transition-transform duration-300 ease-out"
        enter-from-class="translate-x-full"
        enter-to-class="translate-x-0"
        leave-active-class="transition-transform duration-250 ease-in"
        leave-from-class="translate-x-0"
        leave-to-class="translate-x-full"
      >
        <div
          v-if="downloadManager.isOpen.value"
          class="fixed inset-y-0 right-0 z-50 w-full sm:w-[480px] bg-white shadow-2xl flex flex-col overflow-hidden border-l border-gray-100"
        >
          <!-- 1. 顶部 Header -->
          <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between bg-white shrink-0">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded-xl bg-emerald-50 text-emerald-600 flex items-center justify-center shadow-2xs">
                <Download class="w-4 h-4" :class="{ 'animate-bounce': downloadManager.activeCount.value > 0 }" />
              </div>
              <div>
                <h3 class="text-sm font-bold text-gray-800 flex items-center gap-2">
                  <span>下载任务</span>
                  <span
                    v-if="downloadManager.activeCount.value > 0"
                    class="px-2 py-0.5 rounded-full bg-emerald-500 text-white text-[10px] font-semibold animate-pulse"
                  >
                    {{ downloadManager.activeCount.value }} 首下载中
                  </span>
                </h3>
                <p class="text-[11px] text-gray-400">
                  后台持久下载，切换页面不中断
                </p>
              </div>
            </div>

            <button
              @click="downloadManager.close()"
              class="p-1.5 rounded-xl text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
              title="关闭下载面板"
            >
              <X class="w-4 h-4" />
            </button>
          </div>

          <!-- 2. 当前存储目录提示条 -->
          <div class="px-5 py-2.5 bg-gray-50/80 border-b border-gray-100 flex items-center justify-between text-xs text-gray-600 shrink-0">
            <div class="flex items-center gap-1.5 min-w-0">
              <Folder class="w-3.5 h-3.5 text-amber-500 shrink-0" />
              <span class="text-gray-400 shrink-0">存储路径:</span>
              <span class="font-mono text-gray-700 truncate text-[11px]" :title="downloadManager.currentDownloadDir.value">
                {{ downloadManager.currentDownloadDir.value }}
              </span>
            </div>
            <button
              @click="downloadManager.openDirModal()"
              class="text-emerald-600 hover:text-emerald-700 hover:underline shrink-0 text-[11px] font-medium ml-2"
            >
              更改目录
            </button>
          </div>

          <!-- 3. 筛选标签 & 操作栏 -->
          <div class="px-5 py-2 border-b border-gray-100 flex flex-wrap items-center justify-between gap-2 bg-white shrink-0">
            <!-- 标签切换 -->
            <div class="flex items-center gap-1 bg-gray-100 p-0.5 rounded-lg text-xs overflow-x-auto">
              <button
                v-for="tab in filterTabs" :key="tab.id"
                @click="currentFilter = tab.id"
                :class="[
                  'px-2.5 py-1 rounded-md text-[11px] font-medium transition-all whitespace-nowrap',
                  currentFilter === tab.id
                    ? 'bg-white text-emerald-600 shadow-xs font-semibold'
                    : 'text-gray-500 hover:text-gray-700'
                ]"
              >
                {{ tab.name }}
                <span class="opacity-70 ml-0.5 text-[10px]">({{ tab.count }})</span>
              </button>
            </div>

            <!-- 批量控制操作 -->
            <div class="flex items-center gap-1.5">
              <!-- 全部暂停 -->
              <button
                v-if="downloadManager.activeCount.value > 0"
                @click="downloadManager.pauseAll()"
                class="px-2 py-1 rounded-md bg-amber-50 text-amber-600 hover:bg-amber-100 text-[11px] font-medium transition-colors flex items-center gap-1 cursor-pointer"
                title="暂停所有下载中的任务"
              >
                <Pause class="w-3 h-3" />
                <span>全部暂停</span>
              </button>

              <!-- 全部继续 -->
              <button
                v-if="downloadManager.pausedCount.value > 0"
                @click="downloadManager.resumeAll(activeSource?.id)"
                class="px-2 py-1 rounded-md bg-emerald-50 text-emerald-600 hover:bg-emerald-100 text-[11px] font-medium transition-colors flex items-center gap-1 cursor-pointer"
                title="继续所有已暂停的任务"
              >
                <Play class="w-3 h-3" />
                <span>全部继续</span>
              </button>

              <!-- 全部重试 -->
              <button
                v-if="downloadManager.failedCount.value > 0"
                @click="downloadManager.retryAllFailed(activeSource?.id)"
                class="px-2 py-1 rounded-md bg-blue-50 text-blue-600 hover:bg-blue-100 text-[11px] font-medium transition-colors flex items-center gap-1 cursor-pointer"
                title="重新尝试所有失败的任务"
              >
                <RotateCw class="w-3 h-3" />
                <span>重试失败</span>
              </button>

              <!-- 停止全部进行中任务 -->
              <button
                v-if="downloadManager.activeCount.value > 0 || downloadManager.pausedCount.value > 0"
                @click="downloadManager.stopAll()"
                class="px-2 py-1 rounded-md bg-rose-50 text-rose-600 hover:bg-rose-100 text-[11px] font-medium transition-colors flex items-center gap-1 cursor-pointer"
                title="停止并取消所有正在进行与排队中的任务"
              >
                <Square class="w-3 h-3" />
                <span>停止全部</span>
              </button>

              <!-- 清空已完成 -->
              <button
                v-if="downloadManager.completedCount.value > 0"
                @click="downloadManager.clearCompleted()"
                class="px-2 py-1 rounded-md text-gray-400 hover:text-gray-600 hover:bg-gray-100 text-[11px] transition-colors cursor-pointer"
                title="清除所有已完成的记录"
              >
                清空完成
              </button>
            </div>
          </div>

          <!-- 4. 任务列表内容区 -->
          <div class="flex-1 overflow-y-auto p-4 space-y-2.5 bg-gray-50/50">
            <!-- 空状态 -->
            <div
              v-if="!filteredTasks.length"
              class="flex flex-col items-center justify-center py-24 text-center text-gray-400"
            >
              <div class="w-14 h-14 rounded-2xl bg-gray-100 text-gray-300 flex items-center justify-center mb-3">
                <Music2 class="w-7 h-7" />
              </div>
              <p class="text-xs font-medium text-gray-600 mb-1">
                {{ currentFilter === 'all' ? '暂无下载任务' : `暂无${currentFilterName}任务` }}
              </p>
              <p class="text-[11px] text-gray-400 max-w-xs">
                在「发现音乐」中点击单曲或榜单的下载按钮，歌曲将自动加入后台下载队列并保存在 NAS
              </p>
            </div>

            <!-- 任务卡片列表 -->
            <div
              v-for="task in filteredTasks"
              :key="task.id"
              class="bg-white rounded-xl border border-gray-100 p-3 shadow-2xs hover:border-gray-200 transition-all flex items-center gap-3 group"
            >
              <!-- 封面 -->
              <div class="w-10 h-10 rounded-lg overflow-hidden bg-gray-100 shrink-0 relative flex items-center justify-center shadow-2xs">
                <img
                  v-if="task.song.cover"
                  :src="task.song.cover"
                  alt="cover"
                  referrerpolicy="no-referrer"
                  class="w-full h-full object-cover"
                  @error="task.song.cover = ''"
                />
                <span v-else class="text-base">🎵</span>
              </div>

              <!-- 歌曲信息与状态 -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center justify-between gap-1 mb-0.5">
                  <p class="text-xs font-semibold text-gray-800 truncate" :title="task.song.name">
                    {{ task.song.name }}
                  </p>
                  <!-- 平台与音质标签 -->
                  <div class="flex items-center gap-1 shrink-0">
                    <span v-if="task.quality" class="text-[9px] px-1.5 py-0.2 rounded-full bg-emerald-50 text-emerald-700 font-bold font-mono border border-emerald-200">
                      {{ (task.actualFormat || task.quality).toUpperCase() }}
                    </span>
                    <span class="text-[9px] px-1.5 py-0.2 rounded-full bg-gray-100 text-gray-500 font-mono">
                      {{ task.song.source || task.platform || 'kw' }}
                    </span>
                  </div>
                </div>
                <p class="text-[11px] text-gray-400 truncate mb-1" :title="task.song.singer">
                  {{ task.song.singer }}{{ task.song.album ? ' · ' + task.song.album : '' }}
                </p>

                <!-- 状态徽标与提示 -->
                <div class="flex items-center gap-1.5 text-[10px]">
                  <!-- 等待中 -->
                  <span
                    v-if="task.status === 'pending'"
                    class="px-2 py-0.5 rounded-md bg-gray-100 text-gray-500 flex items-center gap-1"
                  >
                    <Clock class="w-3 h-3 text-gray-400" />
                    <span>排队等待中...</span>
                  </span>

                  <!-- 解析直链中 -->
                  <span
                    v-else-if="task.status === 'resolving'"
                    class="px-2 py-0.5 rounded-md bg-blue-50 text-blue-600 flex items-center gap-1"
                  >
                    <Loader2 class="w-3 h-3 animate-spin text-blue-500" />
                    <span>正在解析音源直链...</span>
                  </span>

                  <!-- 下载写入 NAS 中 -->
                  <span
                    v-else-if="task.status === 'downloading'"
                    class="px-2 py-0.5 rounded-md bg-emerald-50 text-emerald-600 flex items-center gap-1 font-medium"
                  >
                    <Loader2 class="w-3 h-3 animate-spin text-emerald-500" />
                    <span>正在下载写入 NAS...</span>
                  </span>

                  <!-- 已暂停 -->
                  <span
                    v-else-if="task.status === 'paused'"
                    class="px-2 py-0.5 rounded-md bg-amber-50 text-amber-600 flex items-center gap-1 font-medium"
                  >
                    <Pause class="w-3 h-3 text-amber-500" />
                    <span>已暂停下载</span>
                  </span>

                  <!-- 下载成功 -->
                  <span
                    v-else-if="task.status === 'success'"
                    class="px-2 py-0.5 rounded-md bg-emerald-50 text-emerald-600 flex items-center gap-1 font-medium"
                    :title="task.qualityNote || ''"
                  >
                    <Check class="w-3 h-3 text-emerald-500" />
                    <span>已保存至 NAS 本地{{ task.actualFormat ? ` (.${task.actualFormat.toLowerCase()})` : '' }}</span>
                    <span v-if="task.qualityNote" class="text-[9px] text-amber-600 font-normal">({{ task.qualityNote }})</span>
                  </span>

                  <!-- 下载失败 -->
                  <span
                    v-else-if="task.status === 'failed'"
                    class="px-2 py-0.5 rounded-md bg-rose-50 text-rose-600 flex items-center gap-1 max-w-[240px] truncate"
                    :title="task.error"
                  >
                    <AlertCircle class="w-3 h-3 text-rose-500 shrink-0" />
                    <span class="truncate">{{ task.error || '下载失败' }}</span>
                  </span>
                </div>
              </div>

              <!-- 右侧操作按钮 -->
              <div class="shrink-0 flex items-center gap-1">
                <!-- 排队或下载中：暂停按钮 -->
                <button
                  v-if="task.status === 'pending' || task.status === 'resolving' || task.status === 'downloading'"
                  @click="downloadManager.pauseTask(task.id)"
                  class="p-1.5 rounded-lg bg-amber-50 hover:bg-amber-100 text-amber-600 transition-colors"
                  title="暂停下载"
                >
                  <Pause class="w-3.5 h-3.5" />
                </button>

                <!-- 已暂停：继续按钮 -->
                <button
                  v-if="task.status === 'paused'"
                  @click="downloadManager.resumeTask(task.id, activeSource?.id)"
                  class="p-1.5 rounded-lg bg-emerald-50 hover:bg-emerald-100 text-emerald-600 transition-colors"
                  title="继续下载"
                >
                  <Play class="w-3.5 h-3.5" />
                </button>

                <!-- 排队中/下载中/已暂停：取消并停止按钮 -->
                <button
                  v-if="task.status === 'pending' || task.status === 'resolving' || task.status === 'downloading' || task.status === 'paused'"
                  @click="downloadManager.cancelTask(task.id)"
                  class="p-1.5 rounded-lg text-gray-400 hover:text-rose-600 hover:bg-rose-50 transition-colors"
                  title="停止并取消此任务"
                >
                  <Square class="w-3.5 h-3.5" />
                </button>

                <!-- 失败任务重试按钮 -->
                <button
                  v-if="task.status === 'failed'"
                  @click="downloadManager.retryTask(task.id, activeSource?.id)"
                  class="p-1.5 rounded-lg bg-emerald-50 hover:bg-emerald-100 text-emerald-600 transition-colors"
                  title="重新下载"
                >
                  <RotateCw class="w-3.5 h-3.5" />
                </button>

                <!-- 移除记录按钮 (已完成或失败) -->
                <button
                  v-if="task.status === 'success' || task.status === 'failed'"
                  @click="downloadManager.removeTask(task.id)"
                  class="p-1.5 rounded-lg text-gray-300 hover:text-gray-500 hover:bg-gray-100 transition-colors"
                  title="移除记录"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- ── NAS 下载目录设置弹窗 ── -->
    <Teleport to="body">
      <div
        v-if="downloadManager.showDirModal.value"
        class="fixed inset-0 z-60 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs"
      >
        <div class="bg-white rounded-2xl max-w-md w-full p-5 shadow-2xl border border-gray-100 space-y-4">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-bold text-gray-800 flex items-center gap-1.5">
              <Folder class="w-4 h-4 text-amber-500" />
              <span>设置歌曲保存到 NAS 目录</span>
            </h3>
            <button
              @click="downloadManager.showDirModal.value = false"
              class="text-gray-400 hover:text-gray-600"
            >
              <X class="w-4 h-4" />
            </button>
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">存储路径：</label>
            <input
              v-model="downloadManager.editDownloadDir.value"
              type="text"
              placeholder="/vol1/Music"
              class="w-full px-3 py-2 rounded-xl bg-gray-50 border border-gray-200 text-xs font-mono text-gray-800 focus:outline-none focus:border-emerald-400"
            />
          </div>

          <!-- 目录浏览器 -->
          <div>
            <div class="flex items-center justify-between text-xs text-gray-400 mb-1.5">
              <span>NAS 存储卷目录浏览：</span>
              <button
                v-if="downloadManager.browseParentPath.value"
                @click="downloadManager.browseDir(downloadManager.browseParentPath.value)"
                class="text-emerald-600 hover:underline flex items-center gap-0.5"
              >
                <span>↑ 上一级</span>
              </button>
            </div>
            <div class="max-h-48 overflow-y-auto border border-gray-100 rounded-xl bg-gray-50 p-2 space-y-1 divide-y divide-gray-100">
              <div
                v-for="folder in downloadManager.browseFolders.value" :key="folder.path"
                @click="downloadManager.browseDir(folder.path)"
                class="py-1.5 px-2 rounded-lg hover:bg-white cursor-pointer flex items-center justify-between text-xs text-gray-700 transition-colors"
              >
                <div class="flex items-center gap-2 truncate">
                  <Folder class="w-3.5 h-3.5 text-amber-400 shrink-0" />
                  <span class="truncate">{{ folder.name }}</span>
                </div>
                <span class="text-[10px] text-gray-400 shrink-0 font-mono">{{ folder.audio_count }} 首</span>
              </div>
              <div v-if="!downloadManager.browseFolders.value.length" class="py-4 text-center text-xs text-gray-400">
                当前目录下无子文件夹
              </div>
            </div>
          </div>

          <div class="flex justify-end gap-2 pt-2">
            <button
              @click="downloadManager.showDirModal.value = false"
              class="px-4 py-1.5 rounded-xl text-xs text-gray-500 hover:bg-gray-100"
            >
              取消
            </button>
            <button
              @click="downloadManager.saveDownloadDir()"
              :disabled="downloadManager.isSavingDir.value"
              class="px-5 py-1.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white font-semibold text-xs shadow-sm disabled:opacity-50"
            >
              {{ downloadManager.isSavingDir.value ? '保存中...' : '确定设为下载目录' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { downloadManager } from '../services/downloadManager'
import {
  Download, X, Folder, RotateCw, Trash2, Check,
  AlertCircle, Loader2, Music2, Clock, Pause, Play, Square
} from 'lucide-vue-next'

const props = defineProps({
  activeSource: Object
})

const currentFilter = ref('all') // 'all' | 'downloading' | 'paused' | 'completed' | 'failed'

const filterTabs = computed(() => [
  { id: 'all', name: '全部', count: downloadManager.totalCount.value },
  { id: 'downloading', name: '下载中', count: downloadManager.activeCount.value },
  { id: 'paused', name: '已暂停', count: downloadManager.pausedCount.value },
  { id: 'completed', name: '已完成', count: downloadManager.completedCount.value },
  { id: 'failed', name: '失败', count: downloadManager.failedCount.value },
])

const currentFilterName = computed(() => {
  const match = filterTabs.value.find(t => t.id === currentFilter.value)
  return match ? match.name : ''
})

const filteredTasks = computed(() => {
  const list = downloadManager.tasks.value
  if (currentFilter.value === 'downloading') {
    return list.filter(t => t.status === 'pending' || t.status === 'resolving' || t.status === 'downloading')
  }
  if (currentFilter.value === 'paused') {
    return list.filter(t => t.status === 'paused')
  }
  if (currentFilter.value === 'completed') {
    return list.filter(t => t.status === 'success')
  }
  if (currentFilter.value === 'failed') {
    return list.filter(t => t.status === 'failed')
  }
  return list
})
</script>
