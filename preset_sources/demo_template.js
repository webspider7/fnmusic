/*!
 * @name 自定义音源开发协议规范示例 (Demo Template)
 * @description 仅供开发者学习音源脚本结构参考，不包含任何商业平台音频解析
 * @version 1.0.0
 * @author Community
 */

/**
 * 极光音乐 / LX Music 音源脚本接口定义
 */
const demoSource = {
  name: "自定义测试音源",
  version: "1.0.0",
  author: "开发者",
  description: "仅用于接口规范演示，用户需自行编写或导入合规音源",
  
  // 支持的平台标识：wy (网易), tx (企鹅), kg (酷狗), kw (酷我)
  supportQualitys: {
    kw: ["128k", "320k", "flac"],
    wy: ["128k", "320k", "flac"]
  },

  /**
   * 音频直链解析方法
   * @param {string} source - 平台标识
   * @param {object} musicInfo - 歌曲元数据 { id, songmid, name, singer, album }
   * @param {string} quality - 音质规格 ('128k' | '320k' | 'flac')
   * @returns {Promise<string>} 返回音频直链 URL
   */
  async getMusicUrl(source, musicInfo, quality) {
    // 开发者可在此对接自己授权的个人音频服务或合规公开 API
    throw new Error("本模板仅用于规范参考，请配置有效合规的音频服务地址。");
  }
};
