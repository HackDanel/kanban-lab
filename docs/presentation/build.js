// Генератор презентации к защите: node build.js (нужен pptxgenjs).
// Цифры берутся из DATA — обновите их по docs/test-report.md перед сборкой.
const path = require('path')
const pptxgen = require('pptxgenjs')
const { applyTheme } = require(process.env.APPLY_THEME || './apply_theme.js')

const DATA = require('./data.json')

const THEME = {
  name: 'Kanban Lab',
  headFontFace: 'Cambria',
  bodyFontFace: 'Calibri',
  colors: {
    dk1: '1F2933',
    lt1: 'FFFFFF',
    dk2: '0B3C49',
    lt2: 'EEF3F5',
    accent1: '0F766E', // бирюзовый — основной
    accent2: 'F59E0B', // янтарный — акцент, как стикер на доске
    accent3: 'DC2626',
    accent4: '2563EB',
    accent5: '16A34A',
    accent6: '7C3AED',
    hlink: '0F766E',
    folHlink: '0B3C49',
  },
}
const HEX = THEME.colors

const pres = new pptxgen()
pres.layout = 'LAYOUT_WIDE' // 13.33 × 7.5
pres.title = 'Kanban-доска и полный цикл тестирования'
pres.author = DATA.author
pres.theme = { headFontFace: THEME.headFontFace, bodyFontFace: THEME.bodyFontFace }
const C = pres.SchemeColor

const W = 13.333
const M = 0.6

pres.defineSlideMaster({
  title: 'TITLE_DARK',
  background: { color: C.text2 },
  objects: [
    { placeholder: { options: { name: 'title', type: 'title', x: M, y: 2.2, w: W - 2 * M, h: 1.6, fontSize: 44, bold: true, color: C.background1, valign: 'bottom' }, text: '' } },
    { placeholder: { options: { name: 'body', type: 'body', x: M, y: 4.0, w: W - 2 * M, h: 1.2, fontSize: 20, color: C.background2, valign: 'top' }, text: '' } },
  ],
})

pres.defineSlideMaster({
  title: 'CONTENT',
  background: { color: C.background1 },
  margin: [0.5, 0.6, 0.6, 0.6],
  objects: [
    { placeholder: { options: { name: 'title', type: 'title', x: M, y: 0.35, w: W - 2 * M, h: 0.9, fontSize: 32, bold: true, color: C.text2, valign: 'middle' }, text: '' } },
  ],
  slideNumber: { x: W - 1.0, y: 7.0, w: 0.5, h: 0.3, fontSize: 10, color: '6B7785', align: 'right' },
})

const notes = (slide, text) => slide.addNotes(text)

function title(slide, text) {
  slide.addText(text, { placeholder: 'title' })
}

// Карточка в стиле Kanban: белый прямоугольник с мягкой тенью и цветной меткой-«чипом».
function card(slide, { x, y, w, h, chip, chipColor = C.accent1, head, body, headSize = 18, bodySize = 14, name }) {
  slide.addShape(pres.shapes.ROUNDED_RECTANGLE, {
    x, y, w, h, rectRadius: 0.08, fill: { color: C.background1 }, line: { color: 'D5DEE3', width: 0.75 },
    shadow: { type: 'outer', color: '9AA8B3', opacity: 0.35, blur: 6, offset: 2, angle: 90 },
    objectName: `${name}-card`,
  })
  let cy = y + 0.25
  if (chip) {
    slide.addText(chip, {
      x: x + 0.25, y: cy, w: Math.min(w - 0.5, 0.16 * chip.length + 0.5), h: 0.32, fontSize: 11, bold: true,
      color: C.background1, fill: { color: chipColor }, align: 'center', valign: 'middle', margin: 0,
      shape: pres.shapes.ROUNDED_RECTANGLE, rectRadius: 0.05, isTextBox: true, objectName: `${name}-chip`,
    })
    cy += 0.5
  }
  if (head) {
    slide.addText(head, { x: x + 0.25, y: cy, w: w - 0.5, h: 0.5, fontSize: headSize, bold: true, color: C.text1, margin: 0, valign: 'top', isTextBox: true, objectName: `${name}-head` })
    cy += 0.55
  }
  if (body) {
    slide.addText(body, { x: x + 0.25, y: cy, w: w - 0.5, h: y + h - cy - 0.2, fontSize: bodySize, color: '3E4C59', margin: 0, valign: 'top', isTextBox: true, paraSpaceAfter: 4, objectName: `${name}-body` })
  }
}

function stat(slide, { x, y, w, value, label, color = C.accent1, name }) {
  slide.addText(value, { x, y, w, h: 1.0, fontSize: 54, bold: true, color, fontFace: THEME.headFontFace, margin: 0, isTextBox: true, objectName: `${name}-value` })
  slide.addText(label, { x, y: y + 1.0, w, h: 0.7, fontSize: 14, color: '3E4C59', margin: 0, valign: 'top', isTextBox: true, objectName: `${name}-label` })
}

const bullets = (items) => items.map((t, i) => ({ text: t, options: { bullet: true, breakLine: i < items.length - 1 } }))

// 1. Титул
pres.addSection({ title: 'Введение' })
let s = pres.addSlide({ masterName: 'TITLE_DARK', sectionTitle: 'Введение' })
s.addText('Kanban-доска и полный цикл тестирования', { placeholder: 'title' })
s.addText(`Лабораторная работа № 16 · Поддержка и тестирование программных модулей\n${DATA.author}`, { placeholder: 'body' })
;[['To Do', C.background2], ['In Progress', C.accent2], ['Done', C.accent1]].forEach(([t, col], i) => {
  s.addText(t, { x: M + i * 1.9, y: 1.2, w: 1.7, h: 0.5, fontSize: 14, bold: true, color: i === 0 ? C.text2 : C.background1, fill: { color: col }, align: 'center', valign: 'middle', shape: pres.shapes.ROUNDED_RECTANGLE, rectRadius: 0.06, isTextBox: true, objectName: `title-chip-${i}` })
})
notes(s, 'Тема: веб-приложение Kanban-доски, аналог Jira, и полный цикл его тестирования — от требований до CI/CD.')

// 2. Задача
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Введение' })
title(s, 'Что требовалось сделать')
;[
  ['Разработать', C.accent1, 'Kanban-доску уровня Jira', 'Проекты, доски, колонки с WIP-лимитами, карточки с исполнителем, метками, комментариями, вложениями и историей; роли и права доступа'],
  ['Протестировать', C.accent2, '8 видов тестирования', 'Модульное, интеграционное, API, E2E, нагрузочное, безопасность, юзабилити, регрессионное'],
  ['Автоматизировать', C.accent4, 'CI/CD', 'Каждый пуш проходит линтеры, тесты, сканеры и выкатывается на staging, из main — на production'],
].forEach(([chip, col, head, body], i) => card(s, { x: M + i * 4.1, y: 1.8, w: 3.85, h: 3.6, chip, chipColor: col, head, body, bodySize: 16, name: `task-${i}` }))
notes(s, 'Три части работы: продукт, тестирование всеми требуемыми видами, автоматизация в пайплайне.')

// 3. Требования
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Введение' })
title(s, 'Требования прослежены до тестов')
stat(s, { x: M, y: 2.4, w: 3.6, value: String(DATA.fr), label: 'функциональных требований, каждое связано с эндпоинтом и тест-кейсом', name: 'fr' })
stat(s, { x: M + 4.1, y: 2.4, w: 3.6, value: String(DATA.nfr), label: 'нефункциональных: отклик, 10 000 карточек, безопасность, покрытие', color: C.accent2, name: 'nfr' })
stat(s, { x: M + 8.2, y: 2.4, w: 3.6, value: String(DATA.testCases), label: 'тест-кейсов: 8 из методички и 22 дополнительных', color: C.accent4, name: 'tc' })
s.addText('Вне объёма: уведомления и восстановление пароля — опциональные по методичке; TC-08 отмечен неприменимым', { x: M, y: 5.4, w: W - 2 * M, h: 0.6, fontSize: 14, italic: true, color: '6B7785', isTextBox: true, objectName: 'scope-note' })
notes(s, 'Матрица трассировки — docs/requirements.md, тест-кейсы — docs/test-cases.md.')

// 4. Стек
pres.addSection({ title: 'Разработка' })
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Разработка' })
title(s, 'Стек: React на клиенте, Go на сервере')
;[
  ['Frontend', 'React, TypeScript, Redux Toolkit, dnd-kit, Material UI'],
  ['Backend', 'Go, chi, pgx + sqlc, goose, JWT, WebSocket'],
  ['Данные', 'PostgreSQL 17, Redis 8'],
  ['Контракт', 'OpenAPI 3 → код сервера и типы клиента'],
  ['Инфраструктура', 'Docker, Nginx, GitHub Actions, Prometheus, Grafana'],
].forEach(([k, v], i) => {
  s.addText(k, { x: M, y: 1.55 + i * 0.95, w: 2.6, h: 0.7, fontSize: 18, bold: true, color: C.accent1, valign: 'middle', margin: 0, isTextBox: true, objectName: `stack-k-${i}` })
  s.addText(v, { x: M + 2.7, y: 1.55 + i * 0.95, w: 4.6, h: 0.7, fontSize: 16, color: C.text1, valign: 'middle', margin: 0, isTextBox: true, objectName: `stack-v-${i}` })
})
card(s, { x: 8.3, y: 1.5, w: 4.4, h: 4.9, chip: 'Почему Go', chipColor: C.accent2, body: bullets([
  'Требование: ответ быстрее 200 мс при 1000 пользователях',
  'Нативный код, горутина на соединение, малый образ (34 МБ)',
  'sqlc: SQL проверяется при сборке, все запросы параметризованы',
  'Плата: больше явного кода, покрытие ветвей — отдельным инструментом',
]), bodySize: 15, name: 'why-go' })
notes(s, 'Рекомендованный стек — Node/Express; заменил сервер на Go ради требования по производительности. Обоснование и альтернативы — STACK.md.')

// 5. Архитектура
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Разработка' })
title(s, 'Архитектура: монолит со строгими слоями')
const box = (t, x, y, w, col, name, h = 0.8) => s.addText(t, { x, y, w, h, fontSize: 15, bold: true, color: C.background1, fill: { color: col }, align: 'center', valign: 'middle', shape: pres.shapes.ROUNDED_RECTANGLE, rectRadius: 0.08, isTextBox: true, objectName: name })
const arrow = (x1, y1, x2, y2, name) => s.addShape(pres.shapes.LINE, { x: x1, y: y1, w: x2 - x1, h: y2 - y1, line: { color: '6B7785', width: 1.5, endArrowType: 'triangle' }, objectName: name })
box('Браузер: React SPA', M, 1.7, 2.6, C.text2, 'arch-browser')
box('Nginx: TLS, статика, прокси', M + 3.3, 1.7, 3.0, C.text2, 'arch-nginx')
arrow(M + 2.6, 2.1, M + 3.3, 2.1, 'arch-a1')
s.addShape(pres.shapes.ROUNDED_RECTANGLE, { x: M + 7.0, y: 1.4, w: 5.1, h: 3.75, rectRadius: 0.08, fill: { color: C.background2 }, line: { color: 'D5DEE3' }, objectName: 'arch-go' })
s.addText('Сервер на Go', { x: M + 7.2, y: 1.5, w: 4.7, h: 0.4, fontSize: 14, bold: true, color: C.text2, margin: 0, isTextBox: true, objectName: 'arch-go-title' })
;['HTTP: контроллеры, валидация по OpenAPI', 'Сервисы: права, WIP-лимиты, история', 'Репозиторий: запросы sqlc', 'WebSocket-хаб'].forEach((t, i) =>
  box(t, M + 7.2, 2.0 + i * 0.75, 4.7, i === 3 ? C.accent2 : C.accent1, `arch-layer-${i}`, 0.6))
arrow(M + 6.3, 2.1, M + 7.0, 2.1, 'arch-a2')
box('PostgreSQL', M + 7.0, 5.6, 2.4, C.accent4, 'arch-pg')
box('Redis', M + 9.7, 5.6, 2.4, C.accent3, 'arch-redis')
box('Prometheus → Grafana', M, 5.6, 3.0, C.accent6, 'arch-prom')
s.addText('Изменения доски рассылаются через Redis pub/sub всем экземплярам сервера, а оттуда — браузерам по WebSocket', { x: M, y: 3.2, w: 6.1, h: 1.2, fontSize: 14, color: '3E4C59', margin: 0, isTextBox: true, objectName: 'arch-note' })
notes(s, 'Зависимости только вниз: контроллер → сервис → репозиторий. Схема — docs/architecture.md.')

// 6. Приложение
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Разработка' })
title(s, 'Доска: перетаскивание, фильтры, WIP-лимит')
s.addImage({ path: path.join(__dirname, 'img/board.png'), x: M, y: 1.4, w: 8.4, h: 4.725, objectName: 'board-shot', altText: 'Доска со столбцами To Do, In Progress и Done' })
s.addText(bullets([
  'Колонка In Progress заполнена (3 / 3) — кнопка добавления недоступна',
  'Карточка: исполнитель, приоритет, дедлайн, метки',
  'Поиск и фильтры по исполнителю, приоритету, меткам',
  'Изменения других пользователей появляются без перезагрузки',
]), { x: 9.3, y: 1.5, w: 3.45, h: 4.6, fontSize: 15, color: C.text1, valign: 'top', paraSpaceAfter: 8, isTextBox: true, objectName: 'board-notes' })
notes(s, 'Скриншот с реального стенда.')

// 7. Решения
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Разработка' })
title(s, 'Три решения, которые проверялись тестами')
;[
  ['WIP-лимит без гонок', C.accent2, 'Перемещение блокирует строку колонки (SELECT … FOR UPDATE) и только потом считает карточки. 10 параллельных запросов в колонку с лимитом 1 — проходит ровно один.'],
  ['Порядок без перенумерации', C.accent1, 'Позиция — дробное число между соседями: перемещение меняет одну строку. Когда промежуток исчерпан, колонка перенумеровывается.'],
  ['Сессия без XSS и CSRF', C.accent4, 'Access JWT живёт 15 минут в памяти. Refresh-токен в httpOnly-cookie меняется при каждом обновлении; повторное предъявление отзывает всю цепочку.'],
].forEach(([head, col, body], i) => card(s, { x: M + i * 4.1, y: 1.8, w: 3.85, h: 3.6, chip: head, chipColor: col, body, bodySize: 16, name: `decision-${i}` }))
notes(s, 'Каждое решение покрыто интеграционным тестом: TestWipLimitUnderConcurrency, TestCardRenumbering, TestRefreshTokenReuseRevokesSession.')

// 8. Пирамида
pres.addSection({ title: 'Тестирование' })
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Тестирование' })
title(s, `${DATA.totalAutomated} автотестов на всех уровнях`)
const levels = DATA.levels
levels.forEach((lv, i) => {
  const w = 3.2 + i * 1.3
  const x = M + (8.4 - w) / 2 // ярусы центрированы в левой части, справа — список инструментов
  s.addText(`${lv.label}: ${lv.count}`, { x, y: 1.45 + i * 0.82, w, h: 0.7, fontSize: 16, bold: true, color: C.background1, fill: { color: lv.color === 'accent2' ? HEX.accent2 : HEX[lv.color] }, align: 'center', valign: 'middle', isTextBox: true, objectName: `level-${i}` })
})
s.addText(bullets(DATA.tools), { x: 9.2, y: 1.5, w: 3.6, h: 4.8, fontSize: 14, color: C.text1, paraSpaceAfter: 6, isTextBox: true, objectName: 'tools' })
notes(s, 'Пирамида: снизу модульные, выше интеграционные, API и E2E. Нагрузочные и security-тесты запускаются по стенду.')

// 9. Покрытие
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Тестирование' })
title(s, 'Покрытие выше порогов на обеих сторонах')
s.addChart(pres.charts.BAR, [
  { name: 'Строки / операторы', labels: ['Клиент', 'Сервер'], values: DATA.coverage.lines },
  { name: 'Ветви', labels: ['Клиент', 'Сервер'], values: DATA.coverage.branches },
], {
  x: M, y: 1.4, w: 7.6, h: 5.0, barDir: 'col', barGapWidthPct: 60,
  chartColors: [HEX.accent1, HEX.accent2], showValue: true, dataLabelPosition: 'outEnd', dataLabelFormatCode: '0"%"',
  dataLabelFontSize: 14, dataLabelFontFace: '+mn-lt', dataLabelColor: HEX.dk1,
  valAxisMinVal: 0, valAxisMaxVal: 100, valAxisLabelFormatCode: '0"%"', valAxisLabelColor: '6B7785', catAxisLabelColor: HEX.dk1,
  valAxisLabelFontFace: '+mn-lt', catAxisLabelFontFace: '+mn-lt', catAxisLabelFontSize: 14,
  valGridLine: { color: 'E1E7EB', size: 0.5 }, catGridLine: { style: 'none' },
  showLegend: true, legendPos: 'b', legendFontFace: '+mn-lt', legendFontSize: 12,
  showTitle: false, objectName: 'coverage-chart',
})
card(s, { x: 8.8, y: 1.5, w: 3.9, h: 4.8, chip: 'Пороги в CI', chipColor: C.accent3, body: bullets([
  '80% строк и 70% ветвей — сборка падает ниже',
  'Клиент: Vitest (v8) даёт оба показателя',
  'Сервер: go test считает операторы, ветви — gobco',
  'Сгенерированный код (OpenAPI, sqlc) не учитывается',
]), bodySize: 15, name: 'thresholds' })
notes(s, 'Цифры из последнего прогона CI; отчёты — артефакты сборки и Codecov.')

// 10. E2E и API
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Тестирование' })
title(s, 'E2E в трёх браузерах и API-сценарии')
stat(s, { x: M, y: 1.6, w: 3.0, value: `${DATA.e2e} × 3`, label: 'сценариев Playwright: Chrome, Firefox, Edge', name: 'e2e' })
stat(s, { x: M, y: 3.6, w: 3.0, value: String(DATA.newmanAssertions), label: `проверок Newman в ${DATA.newmanRequests} запросах`, color: C.accent2, name: 'newman' })
s.addImage({ path: path.join(__dirname, 'img/card.png'), x: 4.2, y: 1.4, w: 8.5, h: 4.78, objectName: 'card-shot', altText: 'Диалог карточки с комментарием' })
notes(s, 'Полный сценарий: регистрация → вход → проект → доска → карточки → перетаскивание → выход. Плюс права доступа, WIP, XSS, вложения, обновление в реальном времени, адаптивность. Трассировки падений сохраняются в CI.')

// 11. Нагрузка
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Тестирование' })
title(s, DATA.load.headline)
DATA.load.stats.forEach((st, i) => stat(s, { x: M + i * 3.1, y: 1.6, w: 2.9, value: st.value, label: st.label, color: i === 0 ? C.accent1 : i === 1 ? C.accent2 : C.accent4, name: `load-${i}` }))
card(s, { x: M, y: 4.0, w: W - 2 * M, h: 2.5, chip: 'Что нашёл нагрузочный тест', chipColor: C.accent3, body: bullets(DATA.load.findings), bodySize: 15, name: 'load-findings' })
notes(s, DATA.load.notes)

// 12. Безопасность
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Тестирование' })
title(s, 'Безопасность: шесть сканеров в каждой сборке')
DATA.security.forEach(([tool, result], i) => {
  const col = i % 2, row = Math.floor(i / 2)
  card(s, { x: M + col * 6.15, y: 1.45 + row * 1.7, w: 5.9, h: 1.5, head: tool, body: result, headSize: 17, bodySize: 14, name: `sec-${i}` })
})
notes(s, 'Защиты: параметризованные запросы, экранирование React и CSP, Bearer-токен плюс double-submit для cookie, bcrypt, rate limit, HTTPS с HSTS.')

// 13. Дефекты
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Тестирование' })
title(s, `Найдено и исправлено ${DATA.bugs.total} дефектов`)
s.addChart(pres.charts.BAR, [{ name: 'Дефекты', labels: DATA.bugs.bySource.map((b) => b[0]), values: DATA.bugs.bySource.map((b) => b[1]) }], {
  x: M, y: 1.4, w: 7.2, h: 5.0, barDir: 'bar', chartColors: [HEX.accent1], showValue: true, dataLabelPosition: 'outEnd',
  dataLabelFontSize: 14, dataLabelFontFace: '+mn-lt', dataLabelColor: HEX.dk1,
  catAxisLabelColor: HEX.dk1, catAxisLabelFontFace: '+mn-lt', catAxisLabelFontSize: 13, valAxisHidden: true,
  valGridLine: { style: 'none' }, catGridLine: { style: 'none' }, showLegend: false, catAxisOrientation: 'maxMin', objectName: 'bugs-chart',
})
card(s, { x: 8.3, y: 1.5, w: 4.4, h: 4.8, chip: 'Самые важные', chipColor: C.accent3, body: bullets(DATA.bugs.highlights), bodySize: 14, name: 'bugs-top' })
notes(s, 'Каждый дефект оформлен по шаблону: docs/bug-reports. У каждого — коммит с исправлением и тест от регрессии.')

// 14. CI/CD
pres.addSection({ title: 'Итоги' })
s = pres.addSlide({ masterName: 'CONTENT', sectionTitle: 'Итоги' })
title(s, 'CI/CD: от пуша до production без ручных шагов')
const stages = ['Lint', 'Unit', 'Build', 'Integration', 'E2E × 3, API', 'Security', 'Staging', 'Production']
stages.forEach((st, i) => {
  const x = M + i * 1.53
  s.addText(st, { x, y: 2.0, w: 1.38, h: 1.1, fontSize: 14, bold: true, color: C.background1, fill: { color: i < 6 ? HEX.accent1 : i === 6 ? HEX.accent2 : HEX.dk2 }, align: 'center', valign: 'middle', shape: pres.shapes.ROUNDED_RECTANGLE, rectRadius: 0.08, isTextBox: true, objectName: `stage-${i}` })
  if (i < stages.length - 1) arrow(x + 1.38, 2.55, x + 1.53, 2.55, `stage-arrow-${i}`)
})
s.addText(bullets([
  'Пуш в develop выкатывает staging, пуш в main — staging, затем production',
  'Деплой по SSH на VPS: образы из GHCR, проверка здоровья, откат на прошлую версию при сбое',
  'Артефакты сборки: покрытие, отчёты Playwright с трассировками, Newman, ZAP',
  'Нагрузочный тест staging — раз в неделю и по кнопке',
]), { x: M, y: 3.7, w: W - 2 * M, h: 2.6, fontSize: 16, color: C.text1, paraSpaceAfter: 8, isTextBox: true, objectName: 'cicd-notes' })
notes(s, '.github/workflows/ci.yml и load.yml.')

// 15. Итоги
s = pres.addSlide({ masterName: 'TITLE_DARK', sectionTitle: 'Итоги' })
s.addText('Итоги', { placeholder: 'title' })
s.addText(DATA.summary, { placeholder: 'body' })
notes(s, 'Демонстрация: вход, создание карточки, перетаскивание в заполненную колонку, обновление во второй вкладке, запуск автотестов и пайплайна.')

;(async () => {
  const out = path.join(__dirname, 'kanban-lab.pptx')
  await pres.writeFile({ fileName: out })
  await applyTheme(out, THEME)
  console.log('written', out)
})()
