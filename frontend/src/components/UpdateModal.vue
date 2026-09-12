<template>
  <Teleport to="body">
    <!-- 遮罩背景 -->
    <Transition
      enter-active-class="transition-opacity duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="modelValue"
        @click="close"
        class="fixed inset-0 z-[9000] bg-black/60 backdrop-blur-xs flex items-center justify-center p-3 sm:p-4"
        style="z-index: 9000;"
      >
        <!-- 弹窗卡片主体 -->
        <div
          @click.stop
          class="w-full max-w-lg bg-white rounded-2xl shadow-2xl border border-gray-100 overflow-hidden flex flex-col max-h-[86vh] my-auto animate-in fade-in zoom-in-95 duration-200 relative z-10"
        >
          <!-- 顶部 Header -->
          <div class="px-5 py-4 border-b border-gray-100 flex items-center justify-between bg-gradient-to-r from-emerald-50/50 to-teal-50/50">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded-xl bg-emerald-500 text-white flex items-center justify-center shadow-xs">
                <Sparkles class="w-4 h-4" />
              </div>
              <div>
                <h3 class="text-sm font-bold text-gray-800 flex items-center gap-2">
                  <span>版本更新检测</span>
                  <span
                    v-if="updateInfo?.has_update"
                    class="px-2 py-0.5 rounded-full bg-rose-500 text-white text-[10px] font-bold animate-pulse"
                  >
                    发现新版本
                  </span>
                </h3>
                <p class="text-[11px] text-gray-500">
                  当前运行版本：v{{ updateInfo?.current_version || currentVersion }}
                </p>
              </div>
            </div>

            <button
              @click="close"
              class="p-1.5 rounded-xl text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
            >
              <X class="w-4 h-4" />
            </button>
          </div>

          <!-- 内容主体 -->
          <div class="p-5 overflow-y-auto space-y-4 text-xs">
            <!-- 检查中状态 -->
            <div v-if="loading" class="py-12 flex flex-col items-center justify-center text-gray-400 space-y-3">
              <Loader2 class="w-8 h-8 animate-spin text-emerald-500" />
              <p class="text-xs text-gray-600 font-medium">正在获取 GitHub Release 最新版本...</p>
              <p class="text-[11px] text-gray-400">检测仓库：webspider7/fnmusic</p>
            </div>

            <!-- 检测完毕：发现新版本 -->
            <div v-else-if="updateInfo?.has_update" class="space-y-3.5">
              <!-- 版本高亮栏 -->
              <div class="p-3.5 rounded-xl bg-emerald-50/80 border border-emerald-100 flex items-center justify-between">
                <div>
                  <div class="flex items-center gap-2">
                    <span class="text-xs font-bold text-emerald-800">最新版本</span>
                    <span class="px-2 py-0.5 rounded-md bg-emerald-500 text-white font-mono font-bold text-[11px]">
                      v{{ updateInfo.latest_version }}
                    </span>
                  </div>
                  <p class="text-[11px] text-emerald-600 mt-0.5">
                    {{ updateInfo.title || ('极光音乐 v' + updateInfo.latest_version) }}
                    <span v-if="updateInfo.published_at" class="text-emerald-500/80 ml-1">
                      · {{ formatDate(updateInfo.published_at) }}
                    </span>
                  </p>
                </div>

                <div class="text-right">
                  <span class="text-[10px] text-gray-400 block font-mono">
                    {{ formatSize(updateInfo.asset_size) }}
                  </span>
                  <span class="text-[10px] text-emerald-600 font-medium">FPK 安装包</span>
                </div>
              </div>

              <!-- 更新日志 -->
              <div>
                <label class="block text-xs font-semibold text-gray-700 mb-1.5 flex items-center gap-1.5">
                  <FileText class="w-3.5 h-3.5 text-gray-400" />
                  <span>更新内容说明：</span>
                </label>
                <div class="p-3 rounded-xl bg-gray-50 border border-gray-100 text-gray-600 text-[11px] leading-relaxed max-h-48 overflow-y-auto whitespace-pre-wrap font-sans">
                  {{ updateInfo.changelog || '开发者暂未附带更新详情说明。' }}
                </div>
              </div>

              <!-- 飞牛更新操作指南提示 -->
              <div class="p-3 rounded-xl bg-amber-50/70 border border-amber-200/60 text-amber-800 space-y-1">
                <div class="flex items-center gap-1.5 font-bold text-[11px]">
                  <Info class="w-3.5 h-3.5 text-amber-500 shrink-0" />
                  <span>飞牛 OS 更新指引</span>
                </div>
                <p class="text-[10px] text-amber-700 leading-relaxed">
                  点击下方「下载 FPK」后，进入飞牛 NAS 桌面 ➜「应用中心」➜ 左下角「手动安装」➜ 选择刚下载的 FPK 即可直接覆盖更新。所有配置与歌曲完全保留！
                </p>
              </div>
            </div>

            <!-- 检测完毕：已是最新版本 -->
            <div v-else class="py-8 flex flex-col items-center justify-center text-center space-y-2">
              <div class="w-12 h-12 rounded-2xl bg-emerald-50 text-emerald-500 flex items-center justify-center mb-1">
                <CheckCircle2 class="w-7 h-7" />
              </div>
              <h4 class="text-sm font-bold text-gray-800">当前已是最新版本</h4>
              <p class="text-xs text-gray-500 max-w-xs">
                当前运行版本 v{{ updateInfo?.current_version || currentVersion }}，GitHub 仓库暂无更高版本发布。
              </p>
              <div class="text-[11px] text-gray-400 pt-1">
                仓库地址：github.com/webspider7/fnmusic
              </div>
            </div>
          </div>

          <!-- 底部 Footer 操作区 -->
          <div class="px-5 py-3.5 bg-gray-50 border-t border-gray-100 flex items-center justify-between gap-2 shrink-0">
            <!-- 重新检查 -->
            <button
              @click="checkUpdate(true)"
              :disabled="loading"
              class="px-3 py-1.5 rounded-xl border border-gray-200 hover:bg-white text-gray-600 text-xs font-medium flex items-center gap-1 transition-all disabled:opacity-50"
              title="强制重新从 GitHub 检测"
            >
              <RotateCw class="w-3.5 h-3.5" :class="{ 'animate-spin': loading }" />
              <span>重新检查</span>
            </button>

            <!-- 右侧按钮群 -->
            <div class="flex items-center gap-2">
              <button
                @click="openGitHubRelease"
                class="px-3 py-1.5 rounded-xl border border-gray-200 hover:bg-white text-gray-600 text-xs font-medium flex items-center gap-1.5 transition-all"
              >
                <ExternalLink class="w-3.5 h-3.5" />
                <span>GitHub 页面</span>
              </button>

              <button
                v-if="updateInfo?.has_update"
                @click="downloadFPK"
                class="px-4 py-1.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-semibold flex items-center gap-1.5 shadow-sm transition-all"
              >
                <Download class="w-3.5 h-3.5" />
                <span>下载新版 FPK</span>
              </button>

              <button
                v-else
                @click="close"
                class="px-4 py-1.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-semibold transition-all"
              >
                知道了
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { AppAPI } from '../api/client'
import {
  Sparkles, X, Loader2, FileText, Info, CheckCircle2,
  RotateCw, ExternalLink, Download
} from 'lucide-vue-next'

const props = defineProps({
  modelValue: Boolean,
})

const emit = defineEmits(['update:modelValue', 'update-available'])

const currentVersion = '1.2.7'
const loading = ref(false)
const updateInfo = ref(null)

function close() {
  emit('update:modelValue', false)
}

async function checkUpdate(force = false) {
  loading.value = true
  try {
    const res = await AppAPI.checkUpdate(force)
    if (res.code === 200 && res.data) {
      updateInfo.value = res.data
      if (res.data.has_update) {
        emit('update-available', res.data)
      }
    }
  } catch (err) {
    console.warn('Check update error:', err)
  } finally {
    loading.value = false
  }
}

function openGitHubRelease() {
  const url = updateInfo.value?.release_url || 'https://github.com/webspider7/fnmusic/releases'
  window.open(url, '_blank', 'noopener,noreferrer')
}

function downloadFPK() {
  const url = updateInfo.value?.download_url || 'https://github.com/webspider7/fnmusic/releases/latest/download/fn-lx-player.fpk'
  // 打开新标签或直接触发下载
  const a = document.createElement('a')
  a.href = url
  a.download = updateInfo.value?.asset_name || 'fn-lx-player.fpk'
  a.target = '_blank'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

function formatDate(isoStr) {
  if (!isoStr) return ''
  try {
    const d = new Date(isoStr)
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  } catch (_) {
    return ''
  }
}

function formatSize(bytes) {
  if (!bytes) return ''
  const mb = bytes / (1024 * 1024)
  return mb.toFixed(1) + ' MB'
}

defineExpose({
  checkUpdate,
  updateInfo,
})
</script>
