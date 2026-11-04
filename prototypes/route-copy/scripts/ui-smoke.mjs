/**
 * 交互冒烟测试（无需安装依赖）：
 *   node scripts/ui-smoke.mjs
 *
 * 自动拉起 headless chromium（需要本机有 chromium / chromium-browser / google-chrome，
 * 也可用 CHROMIUM_BIN 指定），通过 CDP 驱动页面，校验「高级 → 复制路由」全流程。
 * 默认使用 9222 调试端口，可改 CDP_PORT；端口被占用时会直接失败。
 */
import { spawn } from 'node:child_process'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const ROOT = dirname(dirname(fileURLToPath(import.meta.url)))
const FILE_URL = pathToFileURL(join(ROOT, 'index.html')).href
const CDP_PORT = Number(process.env.CDP_PORT || 9222)
const CDP_HTTP = 'http://127.0.0.1:' + CDP_PORT

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

async function checkPortFree() {
  try {
    await fetch(CDP_HTTP + '/json/version', { signal: AbortSignal.timeout(800) })
    throw new Error('调试端口 ' + CDP_PORT + ' 已被占用，请先关闭占用进程或设置 CDP_PORT')
  } catch (error) {
    if (error.name === 'TimeoutError' || error.name === 'TypeError' || /fetch failed/.test(error.message)) return
    throw error
  }
}

async function launchBrowser(userDataDir) {
  if (typeof WebSocket === 'undefined') {
    throw new Error('当前 Node 版本没有内置 WebSocket，请使用 Node >= 22')
  }
  const args = [
    '--headless=new',
    '--no-sandbox',
    '--disable-gpu',
    '--remote-debugging-port=' + CDP_PORT,
    '--user-data-dir=' + userDataDir,
    'about:blank',
  ]
  const candidates = [process.env.CHROMIUM_BIN, 'chromium', 'chromium-browser', 'google-chrome', 'google-chrome-stable'].filter(Boolean)

  for (const bin of candidates) {
    const child = spawn(bin, args, { stdio: 'ignore' })
    const failed = await new Promise((resolve) => {
      const timer = setTimeout(() => resolve(false), 400)
      child.once('error', () => {
        clearTimeout(timer)
        resolve(true)
      })
    })
    if (!failed) return child
    try {
      child.kill()
    } catch {
      /* ignore */
    }
  }
  throw new Error('未找到 chromium / chromium-browser / google-chrome（可用 CHROMIUM_BIN 指定路径）')
}

async function main() {
  await checkPortFree()
  const userDataDir = mkdtempSync(join(tmpdir(), 'route-copy-chrome-'))
  const browser = await launchBrowser(userDataDir)

  let ws
  try {
    let version
    for (let i = 0; i < 60 && !version; i++) {
      try {
        version = await fetch(CDP_HTTP + '/json/version').then((response) => response.json())
      } catch {
        await sleep(150)
      }
    }
    if (!version) throw new Error('chromium 调试端点未就绪')

    const target = await fetch(CDP_HTTP + '/json/new?' + encodeURIComponent(FILE_URL), { method: 'PUT' }).then((response) => response.json())
    ws = new WebSocket(target.webSocketDebuggerUrl)
    await new Promise((resolve, reject) => {
      ws.onopen = resolve
      ws.onerror = () => reject(new Error('CDP WebSocket 连接失败'))
    })

    let nextId = 0
    const pending = new Map()
    const exceptions = []
    ws.onmessage = (event) => {
      const message = JSON.parse(event.data)
      if (message.id && pending.has(message.id)) {
        const { resolve, reject } = pending.get(message.id)
        pending.delete(message.id)
        if (message.error) reject(new Error(JSON.stringify(message.error)))
        else resolve(message.result)
        return
      }
      if (message.method === 'Runtime.exceptionThrown') {
        const detail = message.params.exceptionDetails
        exceptions.push(detail.exception?.description || detail.text)
      }
    }

    const send = (method, params = {}) => {
      const id = ++nextId
      return new Promise((resolve, reject) => {
        pending.set(id, { resolve, reject })
        ws.send(JSON.stringify({ id, method, params }))
      })
    }

    const evaluate = async (expression) => {
      const result = await send('Runtime.evaluate', {
        expression,
        returnByValue: true,
        awaitPromise: true,
        userGesture: true,
      })
      if (result.exceptionDetails) {
        throw new Error('evaluate 失败: ' + (result.exceptionDetails.exception?.description || result.exceptionDetails.text))
      }
      return result.result.value
    }

    const waitFor = async (expression, timeout = 5000) => {
      const start = Date.now()
      while (Date.now() - start < timeout) {
        if (await evaluate(expression)) return true
        await sleep(80)
      }
      return false
    }

    const assert = (condition, label, extra) => {
      if (!condition) throw new Error('断言失败: ' + label + (extra === undefined ? '' : ' → ' + JSON.stringify(extra)))
      console.log('ok  ' + label)
    }

    await send('Runtime.enable')
    await send('Page.enable')
    await sleep(700)

    // 1. 列表渲染
    assert((await evaluate("document.querySelectorAll('#rules-tbody tr').length")) === 7, '列表渲染 7 条规则')

    // 2. 高级菜单 → 复制路由
    await evaluate("document.querySelector('tr[data-id=\"r2\"] [data-menu]').click()")
    assert(await waitFor("!!document.querySelector('.action-menu-panel')"), '高级菜单弹出')
    const menuItems = await evaluate("[...document.querySelectorAll('.action-menu-item')].map((button) => button.textContent)")
    assert(JSON.stringify(menuItems) === JSON.stringify(['复制路由', 'Paths', '详情', '上移', '下移', '删除']), '菜单项（复制置顶）', menuItems)
    await evaluate("[...document.querySelectorAll('.action-menu-item')].find((button) => button.textContent === '复制路由').click()")
    assert(await waitFor("!!document.querySelector('.drawer[aria-labelledby=\"copy-title\"]')"), '复制抽屉打开')

    const title = await evaluate("document.querySelector('#copy-title').textContent.replace(/\\s+/g, ' ').trim()")
    assert(title.includes('复制路由') && title.includes('源规则 #2'), '抽屉标题含源规则信息', title)
    assert((await evaluate("document.querySelector('#copy-host').value")) === 'api-copy.example.com', 'Host 自动生成区分名')

    const chips = await evaluate("[...document.querySelectorAll('.copy-chips .chip')].map((el) => el.textContent)")
    assert(chips.some((chip) => chip.includes('2 条 Path')) && chips.some((chip) => chip.includes('限流 100/60s')), '复制内容清单 chips', chips)

    // 3. 冲突拦截
    await evaluate(`(() => {
      const input = document.querySelector('#copy-host')
      input.value = 'www.example.com'
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })()`)
    assert(await waitFor("!document.querySelector('#copy-host-error').hidden"), '重复 Host 显示错误')
    assert((await evaluate("document.querySelector('#copy-host-error').textContent")).includes('规则 #1'), '错误提示指明冲突规则')
    assert((await evaluate("document.querySelector('#copy-submit').disabled")) === true, '冲突时创建按钮禁用')

    // 4. 改回合法 Host → 创建副本（紧随源规则之后）
    await evaluate(`(() => {
      const input = document.querySelector('#copy-host')
      input.value = 'api-copy.example.com'
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })()`)
    await evaluate("document.querySelector('#copy-submit').click()")
    assert(await waitFor("!document.querySelector('.drawer-overlay') && document.querySelectorAll('#rules-tbody tr').length === 8"), '副本插入列表（8 条）')
    const inserted = await evaluate(`(() => {
      const hosts = [...document.querySelectorAll('#rules-tbody .host-link code')].map((el) => el.textContent)
      return { hosts, index: hosts.indexOf('api-copy.example.com') }
    })()`)
    assert(inserted.index === 2, '副本紧随源规则之后（第 3 条）', inserted)
    assert(await evaluate("!document.querySelector('#dirty-badge').hidden"), '未保存标记出现')
    const toastText = await evaluate("document.querySelector('.toast')?.textContent || ''")
    assert(toastText.includes('撤销'), 'toast 提供撤销', toastText)

    // 5. 撤销
    await evaluate("[...document.querySelectorAll('.toast-action')].find((button) => button.textContent === '撤销').click()")
    assert(await waitFor("document.querySelectorAll('#rules-tbody tr').length === 7"), 'toast 撤销生效')

    // 6. 再复制并保存发布
    await evaluate("document.querySelector('tr[data-id=\"r2\"] [data-menu]').click()")
    await waitFor("!!document.querySelector('.action-menu-panel')")
    await evaluate("[...document.querySelectorAll('.action-menu-item')].find((button) => button.textContent === '复制路由').click()")
    await waitFor("!!document.querySelector('#copy-submit')")
    await evaluate("document.querySelector('#copy-submit').click()")
    await waitFor("document.querySelectorAll('#rules-tbody tr').length === 8")
    await evaluate("document.querySelector('#btn-save').click()")
    assert(await waitFor("!!document.querySelector('[data-overlay=\"save\"]')"), '保存与发布抽屉打开')
    const diffAdds = await evaluate("[...document.querySelectorAll('.diff-line.add')].map((line) => line.textContent)")
    assert(diffAdds.some((line) => line.includes('api-copy.example.com')), 'diff 含新增规则', diffAdds.slice(0, 3))
    await evaluate("document.querySelector('#save-publish').click()")
    assert(await waitFor("!document.querySelector('[data-overlay=\"save\"]')", 6000), '发布完成后抽屉关闭')
    assert(await evaluate("document.querySelector('#dirty-badge').hidden"), '发布后未保存标记消失')
    assert((await evaluate("document.querySelector('#rules-tbody tr:nth-child(3) .chip')")) === null, '发布后副本标记消失')

    // 7. 详情页入口 + 通配符
    await evaluate("document.querySelector('tr[data-id=\"r5\"] .host-link').click()")
    assert(await waitFor("!document.querySelector('#view-detail').hidden"), '路由详情页打开')
    await evaluate("document.querySelector('[data-copy-detail]').click()")
    assert(await waitFor("!!document.querySelector('#copy-host')"), '详情页可打开复制抽屉')
    assert((await evaluate("document.querySelector('#copy-host').value")) === '*.shop-copy.example.com', '通配符副本保留 *. 前缀')

    // 8. 正则 Host：不自动改名 + 冲突拦截
    await send('Page.navigate', { url: FILE_URL + '?reload=regex#copy=r6' })
    await sleep(900)
    assert((await evaluate("document.querySelector('#copy-host').value")) === '^legacy\\.example\\.com$', '正则 Host 不自动改名')
    assert(await waitFor("!document.querySelector('#copy-host-error').hidden"), '正则副本与原规则同 Host 时提示冲突')
    assert((await evaluate("document.querySelector('#copy-submit').disabled")) === true, '正则冲突时禁用创建')

    // 9. 插入末尾 + 仅保存 + Esc 关闭
    await send('Page.navigate', { url: FILE_URL + '?reload=more#copy=r2' })
    await sleep(900)
    await evaluate(`(() => {
      const select = document.querySelector('#copy-position')
      select.value = 'end'
      select.dispatchEvent(new Event('change', { bubbles: true }))
    })()`)
    await evaluate("document.querySelector('#copy-submit').click()")
    assert(await waitFor("document.querySelectorAll('#rules-tbody tr').length === 8"), '末尾插入后共 8 条')
    const lastHost = await evaluate("[...document.querySelectorAll('#rules-tbody .host-link code')].pop().textContent")
    assert(lastHost === 'api-copy.example.com', '插入位置可选列表末尾', lastHost)
    await evaluate("document.querySelector('#btn-save').click()")
    await waitFor("!!document.querySelector('#save-only')")
    await evaluate("document.querySelector('#save-only').click()")
    assert(await waitFor("document.querySelector('#dirty-badge').hidden"), '仅保存后未保存标记消失')
    assert((await evaluate("document.querySelector('#rules-tbody tr:last-child .chip')?.textContent || ''")).includes('已保存'), '仅保存后行标记待发布')

    await evaluate("document.querySelector('tr[data-id=\"r1\"] [data-menu]').click()")
    await waitFor("!!document.querySelector('.action-menu-panel')")
    await evaluate("[...document.querySelectorAll('.action-menu-item')].find((button) => button.textContent === '复制路由').click()")
    assert(await waitFor("!!document.querySelector('.drawer-overlay')"), '再次打开抽屉')
    await evaluate("document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))")
    assert(await waitFor("!document.querySelector('.drawer-overlay')"), 'Esc 关闭抽屉')

    assert(exceptions.length === 0, '无未捕获 JS 异常', exceptions)
    console.log('\nall UI checks passed')
  } finally {
    try {
      ws?.close()
    } catch {
      /* ignore */
    }
    browser.kill('SIGTERM')
    await sleep(300)
    rmSync(userDataDir, { recursive: true, force: true })
  }
}

main().catch((error) => {
  console.error('\nFAILED: ' + error.message)
  process.exitCode = 1
})
